// Command telemetry-receiver is the collection endpoint for Gameplane's
// anonymous usage telemetry. The API's reporter (api/internal/telemetry)
// POSTs a tiny JSON payload — {version, servers, templates}, nothing
// identifying — once a day when the admin has opted in; this receiver
// validates it, logs it structurally, and exposes aggregate Prometheus
// metrics so an operator can chart adoption without storing raw reports.
//
// It is deliberately standalone (own module, stdlib + client_golang
// only) so it can run anywhere: in-cluster via the Helm chart's
// api.telemetry.receiver.enabled, or on a public host collecting reports
// from many installs.
package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Version is set at build time via -ldflags "-X main.Version=...".
var Version = "dev"

// maxBody bounds the inbound POST body. Real payloads are well under a
// kilobyte; this just stops a misdirected client from streaming at us.
const maxBody = 16 << 10

// HTTP server timeouts: the whole-body read is bounded so a client can't
// hold /ingest open after sending a report.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 15 * time.Second
	idleTimeout       = 120 * time.Second
)

// versionRE bounds what lands in the reports_total version label —
// free-form input must not be able to explode label cardinality with
// garbage. Anything else is counted under "invalid".
var versionRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,31}$`)

type config struct {
	listen    string
	authToken string
}

func loadConfig() config {
	return config{
		listen:    envOr("LISTEN_ADDR", ":8080"),
		authToken: envOr("AUTH_TOKEN", ""),
	}
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// payload mirrors api/internal/telemetry's report shape.
type payload struct {
	Version   string
	Servers   int
	Templates int
}

// errInvalidPayload is returned by decodePayload for any body that is not
// exactly one JSON object carrying all three required fields with exact
// case-sensitive key matching and no duplicates.
var errInvalidPayload = errors.New("invalid payload")

// decodePayload parses a body that has already been read in full. It accepts
// exactly one JSON object with version, servers and templates present and no
// other fields; keys are matched case-sensitively and duplicates are rejected;
// anything after that object other than whitespace is rejected.
func decodePayload(body []byte) (payload, error) {
	dec := json.NewDecoder(bytes.NewReader(body))

	// Expect opening brace
	tok, err := dec.Token()
	if err != nil {
		return payload{}, fmt.Errorf("%w: %w", errInvalidPayload, err)
	}
	if tok != json.Delim('{') {
		return payload{}, fmt.Errorf("%w: expected JSON object", errInvalidPayload)
	}

	var p payload
	seen := make(map[string]bool)
	foundVersion, foundServers, foundTemplates := false, false, false

	// Iterate through object key-value pairs
	for dec.More() {
		// Get the key
		tok, err := dec.Token()
		if err != nil {
			return payload{}, fmt.Errorf("%w: %w", errInvalidPayload, err)
		}

		key, ok := tok.(string)
		if !ok {
			return payload{}, fmt.Errorf("%w: expected string key", errInvalidPayload)
		}

		// Check for duplicate keys
		if seen[key] {
			return payload{}, fmt.Errorf("%w: duplicate key %q", errInvalidPayload, key)
		}
		seen[key] = true

		// Decode the value into a RawMessage to validate type carefully
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return payload{}, fmt.Errorf("%w: %w", errInvalidPayload, err)
		}

		// Reject null values
		if bytes.Equal(raw, []byte("null")) {
			return payload{}, fmt.Errorf("%w: null value for key %q", errInvalidPayload, key)
		}

		// Handle each key
		switch key {
		case "version":
			foundVersion = true
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				return payload{}, fmt.Errorf("%w: version must be string: %w", errInvalidPayload, err)
			}
			p.Version = v

		case "servers":
			foundServers = true
			var s int
			if err := json.Unmarshal(raw, &s); err != nil {
				return payload{}, fmt.Errorf("%w: servers must be integer: %w", errInvalidPayload, err)
			}
			p.Servers = s

		case "templates":
			foundTemplates = true
			var t int
			if err := json.Unmarshal(raw, &t); err != nil {
				return payload{}, fmt.Errorf("%w: templates must be integer: %w", errInvalidPayload, err)
			}
			p.Templates = t

		default:
			return payload{}, fmt.Errorf("%w: unknown field %q", errInvalidPayload, key)
		}
	}

	// Expect closing brace
	tok, err = dec.Token()
	if err != nil {
		return payload{}, fmt.Errorf("%w: %w", errInvalidPayload, err)
	}
	if tok != json.Delim('}') {
		return payload{}, fmt.Errorf("%w: expected closing brace", errInvalidPayload)
	}

	// Check that all required fields were present
	if !foundVersion || !foundServers || !foundTemplates {
		return payload{}, fmt.Errorf("%w: missing required field", errInvalidPayload)
	}

	// Check for trailing content
	switch err := dec.Decode(&struct{}{}); {
	case errors.Is(err, io.EOF):
		// exactly one value
	case err != nil:
		return payload{}, fmt.Errorf("%w: trailing content after the report: %w", errInvalidPayload, err)
	default:
		return payload{}, fmt.Errorf("%w: trailing content after the report", errInvalidPayload)
	}

	return p, nil
}

type server struct {
	cfg config
	reg *prometheus.Registry

	reports   *prometheus.CounterVec
	servers   prometheus.Histogram
	templates prometheus.Histogram
}

func newServer(cfg config) *server {
	reg := prometheus.NewRegistry()
	s := &server{
		cfg: cfg,
		reg: reg,
		reports: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "gameplane_telemetry_reports_total",
			Help: "Telemetry reports accepted, by reported Gameplane version.",
		}, []string{"version"}),
		// Fleet-size distributions: the buckets skew small because a
		// typical install is a homelab; the top bucket catches big ones.
		servers: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "gameplane_telemetry_servers",
			Help:    "Distribution of GameServer counts across reports.",
			Buckets: []float64{0, 1, 2, 5, 10, 25, 50, 100, 250},
		}),
		templates: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "gameplane_telemetry_templates",
			Help:    "Distribution of GameTemplate counts across reports.",
			Buckets: []float64{0, 1, 2, 5, 10, 25, 50, 100, 250},
		}),
	}
	reg.MustRegister(s.reports, s.servers, s.templates)
	return s
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})
	mux.Handle("GET /metrics", promhttp.HandlerFor(s.reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("POST /ingest", s.ingest)
	return mux
}

func (s *server) ingest(w http.ResponseWriter, req *http.Request) {
	if s.cfg.authToken != "" {
		got := req.Header.Get("Authorization")
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.cfg.authToken)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}
	// Read the whole body first so the size limit applies to all of it, not
	// just to the part a JSON decoder happens to consume.
	body, err := io.ReadAll(http.MaxBytesReader(w, req.Body, maxBody))
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	p, err := decodePayload(body)
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if p.Servers < 0 || p.Templates < 0 {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	version := p.Version
	if !versionRE.MatchString(version) {
		version = "invalid"
	}

	slog.Info("telemetry report",
		"version", version, "servers", p.Servers, "templates", p.Templates)
	s.reports.WithLabelValues(version).Inc()
	s.servers.Observe(float64(p.Servers))
	s.templates.Observe(float64(p.Templates))
	w.WriteHeader(http.StatusNoContent)
}

func newHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func run(cfg config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return serve(ctx, cfg)
}

func serve(ctx context.Context, cfg config) error {
	s := newServer(cfg)
	srv := newHTTPServer(cfg.listen, s.routes())
	errCh := make(chan error, 1)
	go func() {
		slog.Info("telemetry-receiver listening", "addr", cfg.listen, "version", Version, "auth", cfg.authToken != "")
		errCh <- srv.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		return fmt.Errorf("listen on %s: %w", cfg.listen, err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func main() {
	if err := run(loadConfig()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("telemetry-receiver failed", "err", err)
		os.Exit(1)
	}
}
