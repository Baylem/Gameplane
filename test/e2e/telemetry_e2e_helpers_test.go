//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Helpers for TestTelemetryLifecycle's subtests (spec 022).

const (
	metricReports    = "gameplane_telemetry_reports_total"
	metricExtended   = "gameplane_telemetry_extended_reports_total"
	metricDuplicates = "gameplane_telemetry_duplicates_total"
	metricRefused    = "gameplane_telemetry_refused_total"
	// betaReceiverImage is the receiver released with v0.2.0-beta.8. The
	// release workflow tags images with the git tag (type=ref,event=tag).
	betaReceiverImage = "ghcr.io/valgulnecron/gameplane/telemetry-receiver:v0.2.0-beta.8"
)

// receiverScrape reads the bundled receiver's /metrics on its dashboard port.
type receiverScrape struct {
	port  int
	token string
}

func (r receiverScrape) get(t *testing.T, name, labelFilter string) float64 {
	t.Helper()
	return receiverMetric(t, r.port, r.token, name, labelFilter)
}

// waitForMore waits until count() is greater than it was at the call.
func waitForMore(t *testing.T, count func() float64, timeout time.Duration, what string) {
	t.Helper()
	before := count()
	envInstance.Eventually(t, timeout, func() (bool, string) {
		got := count()
		return got > before, fmt.Sprintf("%s = %v, want more than %v", what, got, before)
	})
}

// telemetryView is the part of GET /admin/telemetry the lifecycle test reads.
type telemetryView struct {
	Destination struct {
		Kind string  `json:"kind"`
		Host *string `json:"host"`
	} `json:"destination"`
	OperatorDisabled bool `json:"operatorDisabled"`
	Consent          struct {
		Basic    bool `json:"basic"`
		Extended bool `json:"extended"`
	} `json:"consent"`
	InstallID *string `json:"installId"`
	Preview   *struct {
		Version   string `json:"version"`
		Servers   int    `json:"servers"`
		Templates int    `json:"templates"`
		Ext       *struct {
			InstallID string `json:"installId"`
		} `json:"ext"`
	} `json:"preview"`
	Status struct {
		LastOutcome      string  `json:"lastOutcome"`
		LastIDRotationAt *string `json:"lastIdRotationAt"`
	} `json:"status"`
}

func getTelemetryView(t *testing.T, cli *APIClient) telemetryView {
	t.Helper()
	resp, body, err := cli.Get("/admin/telemetry")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/telemetry: %v status %v body %s", err, statusOf(resp), body)
	}
	var v telemetryView
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("decode /admin/telemetry %q: %v", body, err)
	}
	return v
}

func putTelemetryConsent(t *testing.T, cli *APIClient, basic, extended bool) {
	t.Helper()
	resp, body, err := cli.Do(http.MethodPut, "/admin/config/telemetry",
		map[string]bool{"sendMetrics": basic, "extended": extended})
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT telemetry {%v,%v}: %v status %v body %s", basic, extended, err, statusOf(resp), body)
	}
}

// resetInstallID calls POST /admin/telemetry/install-id and returns the new ID.
func resetInstallID(t *testing.T, cli *APIClient) string {
	t.Helper()
	resp, body, err := cli.Post("/admin/telemetry/install-id", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("POST install-id: %v status %v body %s", err, statusOf(resp), body)
	}
	var out struct {
		InstallID string `json:"installId"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.InstallID == "" {
		t.Fatalf("decode install-id response %q: %v", body, err)
	}
	return out.InstallID
}

// detachedPortForward opens a port-forward that outlives the calling
// subtest. PortForward binds kubectl to t.Context(), which is cancelled when
// the subtest returns, so a client kept by later subtests would lose its
// tunnel. The caller owns the returned stop func (APIClient.Close).
func detachedPortForward(t *testing.T, ns, target string, remotePort int) (int, func()) {
	t.Helper()
	const attempts = 4
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		local, stop, err := envInstance.tryPortForward(context.Background(), ns, target, remotePort)
		if err == nil {
			return local, stop
		}
		lastErr = err
		time.Sleep(time.Second)
	}
	t.Fatalf("port-forward never became ready (target %s/%s:%d) after %d attempts: %v",
		ns, target, remotePort, attempts, lastErr)
	return 0, nil
}

// reconnectAPIClient gives the admin session a new port-forward after the API
// pod was replaced (a rollout restart or a helm upgrade that changes its
// flags). The session lives in the database and the cookie jar and CSRF token
// are reused, so no login is spent; if the session did not survive, it logs in
// once more.
func reconnectAPIClient(t *testing.T, old *APIClient) *APIClient {
	t.Helper()
	local, stop := detachedPortForward(t, "gameplane-system", "svc/gameplane-api", 80)
	old.Close()
	cli := &APIClient{BaseURL: fmt.Sprintf("http://127.0.0.1:%d", local), CSRF: old.CSRF, HTTP: old.HTTP, stop: stop}
	var last string
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); time.Sleep(time.Second) {
		resp, body, err := cli.Get("/admin/telemetry")
		if err == nil && resp.StatusCode == http.StatusOK {
			return cli
		}
		last = fmt.Sprintf("%v status %v body %s", err, statusOf(resp), body)
		if err == nil && resp.StatusCode == http.StatusUnauthorized {
			break
		}
	}
	t.Logf("session not reusable after the API restart (%s): logging in again", last)
	cli.Close()
	// APIClient logs in over a forward bound to this subtest; keep its
	// session and cookie jar but move it onto a forward that outlives it.
	fresh := envInstance.APIClient(t, telemetryAdminUser, telemetryAdminPass)
	fresh.Close()
	local, stop = detachedPortForward(t, "gameplane-system", "svc/gameplane-api", 80)
	return &APIClient{BaseURL: fmt.Sprintf("http://127.0.0.1:%d", local), CSRF: fresh.CSRF, HTTP: fresh.HTTP, stop: stop}
}

// helmUpgradeReuse runs `helm upgrade --reuse-values` with the given --set
// values on the e2e release and waits for the rollout.
func helmUpgradeReuse(ctx context.Context, t *testing.T, sets ...string) {
	t.Helper()
	args := []string{"upgrade", "gameplane", filepath.Join("..", "..", "charts", "gameplane"),
		"--kube-context", envInstance.Context, "--namespace", "gameplane-system", "--reuse-values"}
	for _, set := range sets {
		args = append(args, "--set", set)
	}
	args = append(args, "--wait", "--timeout", "6m")
	if out, err := exec.CommandContext(ctx, "helm", args...).CombinedOutput(); err != nil {
		t.Fatalf("helm upgrade %v failed: %v\n%s", sets, err, out)
	}
}

// bundledReceiverImage is the image the chart's own receiver runs, so a
// second receiver uses the image this branch built.
func bundledReceiverImage(ctx context.Context, t *testing.T) string {
	t.Helper()
	out, err := envInstance.Kubectl(ctx, "get", "deploy/gameplane-telemetry-receiver",
		"--namespace", "gameplane-system", "-o", "jsonpath={.spec.template.spec.containers[0].image}")
	if err != nil || strings.TrimSpace(out) == "" {
		t.Fatalf("read the bundled receiver image: %v\n%s", err, out)
	}
	return strings.TrimSpace(out)
}

// deployTelemetryReceiver applies a one-replica receiver Deployment and
// Service with a unique name in the release namespace and waits for it. With
// dashboard set it is a current receiver with the dashboard token Secret
// mounted (so /metrics is on port 8081), no per-source limit and an emptyDir
// store; otherwise it is run as an old image does, on port 8080 alone.
func deployTelemetryReceiver(ctx context.Context, t *testing.T, kind, image string, dashboard bool) string {
	t.Helper()
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatalf("random name: %v", err)
	}
	name := "telemetry-e2e-" + kind + "-" + hex.EncodeToString(suffix[:])
	env := "            - { name: LISTEN_ADDR, value: \":8080\" }\n"
	volumes, mounts, ports := "", "", "            - { name: http, containerPort: 8080 }\n"
	svcPorts := "    - { name: http, port: 8080, targetPort: http }\n"
	if dashboard {
		env += "            - { name: DATA_DIR, value: /data }\n" +
			"            - { name: INGEST_SOURCE_DAILY_LIMIT, value: \"0\" }\n" +
			"            - name: DASHBOARD_TOKEN\n" +
			"              valueFrom: { secretKeyRef: { name: " + telemetryDashSecret + ", key: token } }\n"
		volumes = "      volumes:\n        - { name: data, emptyDir: {} }\n"
		mounts = "          volumeMounts:\n            - { name: data, mountPath: /data }\n"
		ports += "            - { name: dashboard, containerPort: 8081 }\n"
		svcPorts += "    - { name: dashboard, port: 8081, targetPort: dashboard }\n"
	}
	manifest := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %[1]s
  namespace: gameplane-system
spec:
  replicas: 1
  selector:
    matchLabels: { app.kubernetes.io/name: %[1]s }
  template:
    metadata:
      labels: { app.kubernetes.io/name: %[1]s }
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 65532
        fsGroup: 65532
        seccompProfile: { type: RuntimeDefault }
      containers:
        - name: receiver
          image: %[2]s
          imagePullPolicy: IfNotPresent
          env:
%[3]s          ports:
%[4]s          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities: { drop: [ALL] }
%[5]s%[6]s---
apiVersion: v1
kind: Service
metadata:
  name: %[1]s
  namespace: gameplane-system
spec:
  selector: { app.kubernetes.io/name: %[1]s }
  ports:
%[7]s`, name, image, env, ports, mounts, volumes, svcPorts)

	if out, err := envInstance.KubectlWithStdin(ctx, manifest, "apply", "-f", "-"); err != nil {
		t.Fatalf("apply receiver %s: %v\n%s", name, err, out)
	}
	t.Cleanup(func() {
		_, _ = envInstance.Kubectl(context.Background(), "delete", "deploy/"+name, "svc/"+name,
			"--namespace", "gameplane-system", "--ignore-not-found")
	})
	if out, err := envInstance.Kubectl(ctx, "rollout", "status", "deploy/"+name,
		"--namespace", "gameplane-system", "--timeout=3m"); err != nil {
		t.Fatalf("wait for receiver %s: %v\n%s", name, err, out)
	}
	return name
}

// dashboardRequest sends one request to 127.0.0.1:<port> without following
// redirects, so a 303 can be asserted. headers are added as given (the
// session cookie is Secure and the port-forward is plain HTTP, so a client
// with a jar would never send it back); form, when non-nil, is the POST body.
func dashboardRequest(t *testing.T, port int, method, path string, headers map[string]string, form url.Values) (*http.Response, []byte) {
	t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(t.Context(), method, fmt.Sprintf("http://127.0.0.1:%d%s", port, path), body)
	if err != nil {
		t.Fatalf("new request %s %s: %v", method, path, err)
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{
		Timeout:       15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	return resp, raw
}

// dashboardLoginPage returns the dashboard's /login page, retrying while the
// dashboard listener starts.
func dashboardLoginPage(t *testing.T, port int) []byte {
	t.Helper()
	var page []byte
	envInstance.Eventually(t, time.Minute, func() (bool, string) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/login", port), nil)
		if err != nil {
			return false, err.Error()
		}
		resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
		if err != nil {
			return false, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		page, _ = io.ReadAll(resp.Body)
		return resp.StatusCode == http.StatusOK, fmt.Sprintf("GET /login = %d", resp.StatusCode)
	})
	return page
}

// dashboardLogin signs in with the dashboard token (the form POST needs a
// same-origin Origin header) and returns the Cookie header value to send.
func dashboardLogin(t *testing.T, port int, token string) string {
	t.Helper()
	resp, body := dashboardRequest(t, port, http.MethodPost, "/login",
		map[string]string{"Origin": fmt.Sprintf("http://127.0.0.1:%d", port)}, url.Values{"token": {token}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST /login = %d, want 303; body %s", resp.StatusCode, body)
	}
	for _, c := range resp.Cookies() {
		if c.Name == "gp_telemetry_session" && c.Value != "" {
			return c.Name + "=" + c.Value
		}
	}
	t.Fatal("POST /login set no gp_telemetry_session cookie")
	return ""
}

// dashboardViews returns the views' asOf day and raw JSON, read with the
// dashboard token.
func dashboardViews(t *testing.T, port int) (asOf string, raw []byte) {
	t.Helper()
	resp, body := dashboardRequest(t, port, http.MethodGet, "/api/v1/views",
		map[string]string{"Authorization": "Bearer " + telemetryDashToken}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/views = %d: %s", resp.StatusCode, body)
	}
	var v struct {
		AsOf string `json:"asOf"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("decode views %q: %v", body, err)
	}
	return v.AsOf, body
}

// requireViewsUnchanged fails when the views differ from before. If the UTC
// day rolled over in between, asOf moved and the comparison is skipped.
func requireViewsUnchanged(t *testing.T, port int, asOf string, before []byte) {
	t.Helper()
	nowAsOf, after := dashboardViews(t, port)
	if nowAsOf == asOf && !bytes.Equal(before, after) {
		t.Fatalf("a refused report changed the views (SC-014):\nbefore %s\nafter  %s", before, after)
	}
}

// keepClearOfUTCMidnight sleeps past the next UTC midnight when it is closer
// than within, so that two reports meant for one day cannot straddle two.
func keepClearOfUTCMidnight(t *testing.T, within time.Duration) {
	t.Helper()
	now := time.Now().UTC()
	midnight := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	if midnight.Sub(now) < within {
		time.Sleep(midnight.Sub(now) + 2*time.Second)
	}
}
