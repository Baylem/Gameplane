//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	telemetryAdminUser = "telemetryadmin"
	telemetryAdminPass = "telemetry-admin-pw-1"
	// telemetryDashSecret holds the receiver's dashboard token; the chart
	// mounts it as DASHBOARD_TOKEN and serves /metrics on port 8081 behind it
	// (spec 022 Q8, T067).
	telemetryDashSecret = "telemetry-dashboard"
	telemetryDashToken  = "e2e-telemetry-dashboard-token"
	// telemetryInterval is api.telemetry.interval for this bucket's cluster.
	telemetryInterval = time.Minute
)

// TestTelemetryLifecycle walks the spec 022 US1 path on the bucket's own
// cluster (research R17): a fresh install has a pending notice, "seen" opens
// the gate and the first report reaches the bundled receiver, "all-off" stops
// reports, turning telemetry back on resumes them, and an API restart does
// not bring a report forward. The subtests share state, so they run in order.
//
// It is the bucket's only test because it runs `helm upgrade` on the cluster.
// Admin logins: one (the test never restarts into a second session).
func TestTelemetryLifecycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// ---- setup: dashboard token, chart values, one admin ------------------

	if out, err := envInstance.Kubectl(ctx, "create", "secret", "generic", telemetryDashSecret,
		"--namespace", "gameplane-system", "--from-literal=token="+telemetryDashToken); err != nil {
		t.Fatalf("create dashboard token secret: %v\n%s", err, out)
	}
	helmUpgradeTelemetry(ctx, t)
	envInstance.BootstrapAdmin(t, telemetryAdminUser, telemetryAdminPass)

	// The receiver's dashboard port carries /metrics. It is not restarted
	// again, so one port-forward serves the whole test.
	metricsPort, stopMetrics := envInstance.PortForward(t, "gameplane-system", "svc/gameplane-telemetry-receiver", 8081)
	defer stopMetrics()
	reports := func() float64 { return receiverReportsTotal(t, metricsPort) }

	// The bucket's one admin login. Logging in does not show the notice:
	// only the notice POST (the dashboard rendering it) records that.
	cli := envInstance.APIClient(t, telemetryAdminUser, telemetryAdminPass)
	defer cli.Close()

	t.Run("notice_pending_on_fresh_install", func(t *testing.T) {
		resp, body, err := cli.Get("/admin/telemetry/notice")
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("GET notice: %v status %v body %s", err, statusOf(resp), body)
		}
		var n struct {
			Pending     bool `json:"pending"`
			Destination struct {
				Kind string `json:"kind"`
			} `json:"destination"`
		}
		if err := json.Unmarshal(body, &n); err != nil {
			t.Fatalf("decode notice: %v\n%s", err, body)
		}
		if !n.Pending || n.Destination.Kind != "bundled" {
			t.Fatalf("notice = %s, want pending with the bundled destination", body)
		}
		// Nothing may have been sent before the notice was shown (FR-004).
		if got := reports(); got != 0 {
			t.Fatalf("reports before the notice = %v, want 0", got)
		}
	})

	t.Run("seen_opens_gate_and_first_report_arrives", func(t *testing.T) {
		postNoticeAction(t, cli, "seen", http.StatusNoContent)
		envInstance.Eventually(t, 3*telemetryInterval, func() (bool, string) {
			got := reports()
			return got >= 1, fmt.Sprintf("reports_total = %v, want at least 1", got)
		})
	})

	t.Run("all_off_stops_reports", func(t *testing.T) {
		postNoticeAction(t, cli, "all-off", http.StatusNoContent)
		time.Sleep(20 * time.Second) // let a report already in flight land
		settled := reports()
		envInstance.Consistently(t, 3*telemetryInterval, 10*time.Second, func() (bool, string) {
			got := reports()
			return got == settled, fmt.Sprintf("reports_total = %v, want it to stay at %v after all-off", got, settled)
		})

		// Turn telemetry back on through the settings route.
		resp, body, err := cli.Do(http.MethodPut, "/admin/config/telemetry",
			map[string]bool{"sendMetrics": true, "extended": true})
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("PUT telemetry on: %v status %v body %s", err, statusOf(resp), body)
		}
		envInstance.Eventually(t, 3*telemetryInterval, func() (bool, string) {
			got := reports()
			return got > settled, fmt.Sprintf("reports_total = %v, want more than %v after turning telemetry back on", got, settled)
		})
	})

	t.Run("api_restart_does_not_resend_early", func(t *testing.T) {
		// Wait for a report and note when it was first seen.
		seen := reports()
		sawAt := time.Now()
		envInstance.Eventually(t, 3*telemetryInterval, func() (bool, string) {
			got := reports()
			if got > seen {
				seen, sawAt = got, time.Now()
				return true, ""
			}
			return false, fmt.Sprintf("reports_total = %v, waiting for a report", got)
		})

		if out, err := envInstance.Kubectl(ctx, "rollout", "restart", "deploy/gameplane-api",
			"--namespace", "gameplane-system"); err != nil {
			t.Fatalf("restart API: %v\n%s", err, out)
		}
		if out, err := envInstance.Kubectl(ctx, "rollout", "status", "deploy/gameplane-api",
			"--namespace", "gameplane-system", "--timeout=3m"); err != nil {
			t.Fatalf("wait for API rollout: %v\n%s", err, out)
		}

		// The schedule is persisted: the restarted API reports one interval
		// after the previous report, not at once. Allow 5s for the two
		// one-second polling observations.
		envInstance.Eventually(t, 4*telemetryInterval, func() (bool, string) {
			got := reports()
			return got > seen, fmt.Sprintf("reports_total = %v, waiting for the next report", got)
		})
		if gap := time.Since(sawAt); gap < telemetryInterval-5*time.Second {
			t.Fatalf("next report arrived %v after the previous one, want at least %v (the API restart must not send early)",
				gap, telemetryInterval)
		}
	})
}

// helmUpgradeTelemetry reconfigures the e2e release for this bucket: a 1m
// report interval, the dashboard token, the public summary, and no per-source
// ingest limit (every report here comes from one address).
func helmUpgradeTelemetry(ctx context.Context, t *testing.T) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "helm", "upgrade", "gameplane",
		filepath.Join("..", "..", "charts", "gameplane"),
		"--kube-context", envInstance.Context,
		"--namespace", "gameplane-system",
		"--reuse-values",
		"--set", "api.telemetry.interval=1m",
		"--set", "api.telemetry.receiver.dashboard.tokenSecretRef.name="+telemetryDashSecret,
		"--set", "api.telemetry.receiver.publicSummary.enabled=true",
		"--set", "api.telemetry.receiver.ingestSourceDailyLimit=0",
		"--wait", "--timeout", "6m",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("helm upgrade for the telemetry bucket failed: %v\n%s", err, out)
	}
}

// postNoticeAction POSTs one notice action and requires the given status.
func postNoticeAction(t *testing.T, cli *APIClient, action string, want int) {
	t.Helper()
	resp, body, err := cli.Post("/admin/telemetry/notice", map[string]string{"action": action})
	if err != nil || resp.StatusCode != want {
		t.Fatalf("POST notice %q: %v status %v body %s, want %d", action, err, statusOf(resp), body, want)
	}
}

// statusOf returns the response status, or 0 for a failed request.
func statusOf(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

// receiverReportsTotal sums gameplane_telemetry_reports_total over every
// version label, read from the receiver's dashboard port with the dashboard
// token. It fails the test when /metrics is unreachable.
func receiverReportsTotal(t *testing.T, port int) float64 {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/metrics", port), nil)
	if err != nil {
		t.Fatalf("new metrics request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+telemetryDashToken)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("GET receiver /metrics: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET receiver /metrics = %d: %s", resp.StatusCode, body)
	}
	var total float64
	for _, line := range strings.Split(string(body), "\n") {
		if !strings.HasPrefix(line, "gameplane_telemetry_reports_total") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err != nil {
			t.Fatalf("parse metric line %q: %v", line, err)
		}
		total += v
	}
	return total
}
