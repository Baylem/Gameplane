package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// generateTestTLSCert returns a self-signed ECDSA certificate and key valid
// for 127.0.0.1, usable as both the test collector's server certificate and
// (via its DER bytes) the client's trusted root.
func generateTestTLSCert(t *testing.T) tls.Certificate {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:         true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("build key pair: %v", err)
	}
	return cert
}

// waitForwarderSeesClose polls the forwarder's alive() status for up to 2 seconds,
// returning true if it detects a closed connection within that time.
// This replaces fixed sleeps when waiting for a collector close to reach the socket.
func waitForwarderSeesClose(t *testing.T, f *forwarder) bool {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		f.mu.Lock()
		ok := f.alive()
		f.mu.Unlock()
		if !ok {
			return true // Close detected
		}
		if time.Now().After(deadline) {
			return false // Timeout
		}
		time.Sleep(time.Millisecond) // Small poll interval
	}
}

func TestBuildSyslog(t *testing.T) {
	ts := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	got := buildSyslog(16*8+6, ts, "host1", "gameplane-audit", `{"actor":"alice"}`)
	want := `<134>1 2026-06-30T12:00:00.000Z host1 gameplane-audit - - - {"actor":"alice"}`
	if got != want {
		t.Errorf("buildSyslog =\n %q\nwant\n %q", got, want)
	}
}

func TestBuildSyslog_DefaultsAndSingleLine(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	got := buildSyslog(13, ts, "", "", "line1\nline2\rline3")
	// Empty host/app become "-"; embedded newlines/CRs collapse to spaces.
	want := `<13>1 2026-01-02T03:04:05.000Z - - - - - line1 line2 line3`
	if got != want {
		t.Errorf("buildSyslog =\n %q\nwant\n %q", got, want)
	}
}

func TestFrameFor(t *testing.T) {
	msg := "<134>1 ... hello"
	if got := string(frameFor("tcp", msg)); got != strconv.Itoa(len(msg))+" "+msg {
		t.Errorf("tcp frame = %q", got)
	}
	if got := string(frameFor("udp", msg)); got != msg {
		t.Errorf("udp frame = %q", got)
	}
}

func TestNewServer_Validation(t *testing.T) {
	base := func() config {
		return config{
			syslogAddr: "127.0.0.1:514", network: "tcp",
			appName: "x", facility: "local0", severity: "info",
			dialTimeout: time.Second,
		}
	}
	t.Run("ok computes pri", func(t *testing.T) {
		s, err := newServer(base())
		if err != nil {
			t.Fatalf("newServer: %v", err)
		}
		if s.pri != 16*8+6 { // local0(16), info(6)
			t.Errorf("pri = %d, want %d", s.pri, 16*8+6)
		}
	})
	t.Run("missing addr", func(t *testing.T) {
		c := base()
		c.syslogAddr = ""
		if _, err := newServer(c); err == nil {
			t.Error("want error for missing syslog addr")
		}
	})
	t.Run("bad network", func(t *testing.T) {
		c := base()
		c.network = "sctp"
		if _, err := newServer(c); err == nil {
			t.Error("want error for bad network")
		}
	})
	t.Run("tls requires tcp", func(t *testing.T) {
		c := base()
		c.network = "udp"
		c.useTLS = true
		if _, err := newServer(c); err == nil {
			t.Error("want error for tls over udp")
		}
	})
	t.Run("bad facility", func(t *testing.T) {
		c := base()
		c.facility = "nope"
		if _, err := newServer(c); err == nil {
			t.Error("want error for bad facility")
		}
	})
	t.Run("bad severity", func(t *testing.T) {
		c := base()
		c.severity = "nope"
		if _, err := newServer(c); err == nil {
			t.Error("want error for bad severity")
		}
	})
	t.Run("app name with space rejected", func(t *testing.T) {
		c := base()
		c.appName = "gameplane audit"
		if _, err := newServer(c); err == nil {
			t.Error("want error for APP_NAME containing a space")
		}
	})
	t.Run("app name too long rejected", func(t *testing.T) {
		c := base()
		c.appName = strings.Repeat("a", 49)
		if _, err := newServer(c); err == nil {
			t.Error("want error for APP_NAME over 48 bytes")
		}
	})
	t.Run("app name at max length ok", func(t *testing.T) {
		c := base()
		c.appName = strings.Repeat("a", 48)
		if _, err := newServer(c); err != nil {
			t.Errorf("newServer: %v", err)
		}
	})
	t.Run("empty hostname ok", func(t *testing.T) {
		c := base()
		c.hostname = ""
		if _, err := newServer(c); err != nil {
			t.Errorf("newServer: %v", err)
		}
	})
	t.Run("empty app name ok (renders as -)", func(t *testing.T) {
		c := base()
		c.appName = ""
		if _, err := newServer(c); err != nil {
			t.Errorf("newServer: %v", err)
		}
	})
	t.Run("hostname with space rejected", func(t *testing.T) {
		c := base()
		c.hostname = "my host"
		if _, err := newServer(c); err == nil {
			t.Error("want error for SYSLOG_HOSTNAME containing a space")
		}
	})
	t.Run("hostname too long rejected", func(t *testing.T) {
		c := base()
		c.hostname = strings.Repeat("h", 256)
		if _, err := newServer(c); err == nil {
			t.Error("want error for SYSLOG_HOSTNAME over 255 bytes")
		}
	})
	t.Run("hostname with non-printable rejected", func(t *testing.T) {
		c := base()
		c.hostname = "host\x01name"
		if _, err := newServer(c); err == nil {
			t.Error("want error for SYSLOG_HOSTNAME containing a non-printable byte")
		}
	})
}

// tcpSyslogSink starts a one-shot TCP listener that returns the first frame it
// reads. Returns its address and a channel delivering the received bytes.
func tcpSyslogSink(t *testing.T) (string, <-chan string) {
	t.Helper()
	lc := &net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	got := make(chan string, 1)
	go func() {
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		n, _ := conn.Read(buf)
		got <- string(buf[:n])
	}()
	return ln.Addr().String(), got
}

func TestHandle_ForwardsToTCPSyslog(t *testing.T) {
	addr, got := tcpSyslogSink(t)
	s, err := newServer(config{
		syslogAddr: addr, network: "tcp", appName: "gameplane-audit",
		facility: "local0", severity: "info", hostname: "h", dialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	srv := httptest.NewServer(s.routes())
	defer srv.Close()

	body := strings.NewReader(`{"actor":"admin","path":"/api/v1/servers"}`)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/", body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}

	select {
	case frame := <-got:
		// Octet-counting prefix, RFC 5424 header, and the JSON payload.
		if !strings.Contains(frame, "<134>1 ") {
			t.Errorf("missing PRI/version header: %q", frame)
		}
		if !strings.Contains(frame, "gameplane-audit") {
			t.Errorf("missing app-name: %q", frame)
		}
		if !strings.Contains(frame, `{"actor":"admin","path":"/api/v1/servers"}`) {
			t.Errorf("missing JSON payload: %q", frame)
		}
		prefix := frame[:strings.IndexByte(frame, ' ')]
		if _, err := strconv.Atoi(prefix); err != nil {
			t.Errorf("octet-count prefix not numeric: %q", prefix)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("syslog sink received nothing")
	}
}

func TestHandle_ForwardsToUDPSyslog(t *testing.T) {
	lc := &net.ListenConfig{}
	pc, err := lc.ListenPacket(context.Background(), "udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	defer pc.Close()
	got := make(chan string, 1)
	go func() {
		buf := make([]byte, 4096)
		n, _, err := pc.ReadFrom(buf)
		if err == nil {
			got <- string(buf[:n])
		}
	}()

	s, err := newServer(config{
		syslogAddr: pc.LocalAddr().String(), network: "udp", appName: "gp",
		facility: "local0", severity: "info", hostname: "h", dialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(`{"k":"v"}`))
	s.handle(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
	select {
	case msg := <-got:
		// UDP carries the bare message (no octet-count prefix).
		if !strings.HasPrefix(msg, "<134>1 ") || !strings.Contains(msg, `{"k":"v"}`) {
			t.Errorf("udp message = %q", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("udp sink received nothing")
	}
}

func TestHandle_AuthRequired(t *testing.T) {
	s := &server{authHeader: "Bearer secret", network: "tcp", fwd: newForwarder("tcp", "127.0.0.1:1", false, time.Second)}

	t.Run("missing token", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(`{}`))
		s.handle(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rr.Code)
		}
	})
	t.Run("wrong token", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer nope")
		s.handle(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rr.Code)
		}
	})
}

func TestHandle_MethodAndBody(t *testing.T) {
	s := &server{network: "tcp", fwd: newForwarder("tcp", "127.0.0.1:1", false, time.Second)}

	t.Run("GET rejected", func(t *testing.T) {
		rr := httptest.NewRecorder()
		s.handle(rr, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", rr.Code)
		}
	})
	t.Run("empty body", func(t *testing.T) {
		rr := httptest.NewRecorder()
		s.handle(rr, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader("   ")))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})
}

func TestHandle_ForwardFailureIs502(t *testing.T) {
	// Point at a closed port so the dial fails.
	s := &server{network: "tcp", appName: "x", hostname: "h",
		fwd: newForwarder("tcp", "127.0.0.1:1", false, 200*time.Millisecond)}
	rr := httptest.NewRecorder()
	s.handle(rr, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(`{"k":"v"}`)))
	if rr.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", rr.Code)
	}
}

func TestForwarder_ReusesConnection(t *testing.T) {
	lc := &net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	// Accept exactly one connection and accumulate everything read on it. Both
	// sends must arrive on this same connection — if the forwarder opened a
	// second one for "two", it would sit unaccepted and the test would time out,
	// so this proves connection reuse. Accumulating (rather than counting reads)
	// is robust to TCP coalescing "one"+"two" into a single Read.
	got := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		var acc []byte
		tmp := make([]byte, 256)
		for {
			n, err := conn.Read(tmp)
			acc = append(acc, tmp[:n]...)
			if strings.Contains(string(acc), "one") && strings.Contains(string(acc), "two") {
				got <- string(acc)
				return
			}
			if err != nil {
				return
			}
		}
	}()

	f := newForwarder("tcp", ln.Addr().String(), false, time.Second)
	if err := f.send(context.Background(), []byte("one")); err != nil {
		t.Fatalf("send one: %v", err)
	}
	if err := f.send(context.Background(), []byte("two")); err != nil {
		t.Fatalf("send two: %v", err)
	}
	select {
	case s := <-got:
		if !strings.Contains(s, "one") || !strings.Contains(s, "two") {
			t.Errorf("received %q, want both frames", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive both frames on a reused connection")
	}
}

// TestForwarder_ReconnectsAfterWriteFailure exercises the redial-and-retry
// branch in forwarder.send: a write that fails on the held connection must
// close it, dial a fresh one, and retry the same frame.
func TestForwarder_ReconnectsAfterWriteFailure(t *testing.T) {
	lc := &net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	type accepted struct {
		n    int
		data string
	}
	acceptCh := make(chan accepted, 2)
	go func() {
		for i := 0; i < 2; i++ {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			// Bound the read so a missing frame can't park this goroutine.
			_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			buf := make([]byte, 256)
			n, _ := conn.Read(buf)
			acceptCh <- accepted{n: i + 1, data: string(buf[:n])}
			// Leave the connection open; the client side (not this accepted
			// conn) is what gets force-closed to simulate the dead link.
			defer conn.Close()
		}
	}()

	f := newForwarder("tcp", ln.Addr().String(), false, time.Second)
	if err := f.send(context.Background(), []byte("one")); err != nil {
		t.Fatalf("send one: %v", err)
	}
	select {
	case got := <-acceptCh:
		if got.n != 1 || got.data != "one" {
			t.Fatalf("first accept = %+v, want frame \"one\"", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first connection never received \"one\"")
	}

	// Force the held connection dead from the client side, so the next send
	// must observe a write failure and go through dial+retry.
	f.mu.Lock()
	_ = f.conn.Close()
	f.mu.Unlock()

	if err := f.send(context.Background(), []byte("two")); err != nil {
		t.Fatalf("send two (post-reconnect): %v", err)
	}
	select {
	case got := <-acceptCh:
		if got.n != 2 || got.data != "two" {
			t.Fatalf("second accept = %+v, want a fresh connection carrying \"two\"", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("forwarder did not reconnect and deliver \"two\" on a new connection")
	}
}

// TestForwarder_WriteDeadlineExceeded exercises the deadline set in
// forwarder.write: a collector that accepts the connection but never drains
// it must make Write fail once dialTimeout elapses, instead of blocking
// forever.
//
// The forwarder dials lazily inside send, so the test must not wait for an
// Accept before calling send. A background loop accepts (and holds, unread)
// every connection until cleanup closes the listener and the held conns, and
// send runs in its own goroutine so every wait is bounded by a timer.
func TestForwarder_WriteDeadlineExceeded(t *testing.T) {
	lc := &net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	var (
		heldMu sync.Mutex
		held   []net.Conn
	)
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed by cleanup
			}
			// Deliberately never Read: the kernel buffers fill and the
			// forwarder's Write blocks until its deadline fires.
			heldMu.Lock()
			held = append(held, conn)
			heldMu.Unlock()
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		select {
		case <-acceptDone:
		case <-time.After(5 * time.Second):
			t.Error("accept loop did not exit after listener close")
		}
		heldMu.Lock()
		for _, c := range held {
			_ = c.Close()
		}
		heldMu.Unlock()
	})

	f := newForwarder("tcp", ln.Addr().String(), false, 200*time.Millisecond)
	// Far larger than loopback send+receive socket buffers combined, so the
	// write cannot complete while the peer never reads.
	frame := make([]byte, 64<<20)

	errCh := make(chan error, 1)
	start := time.Now()
	go func() { errCh <- f.send(context.Background(), frame) }()

	select {
	case err = <-errCh:
	case <-time.After(10 * time.Second):
		t.Fatal("send did not return; write deadline was not enforced")
	}
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("want a write-deadline error, got nil")
	}
	var nerr net.Error
	if !errors.Is(err, os.ErrDeadlineExceeded) && (!errors.As(err, &nerr) || !nerr.Timeout()) {
		t.Errorf("err = %v, want a deadline-exceeded / timeout error", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("send took %s, want it bounded by the ~200ms write deadline (x2 for the retry)", elapsed)
	}
}

func TestHealthz(t *testing.T) {
	s := &server{network: "tcp", fwd: newForwarder("tcp", "127.0.0.1:1", false, time.Second)}
	rr := httptest.NewRecorder()
	s.routes().ServeHTTP(rr, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("healthz status = %d", rr.Code)
	}
	b, _ := io.ReadAll(rr.Body)
	if string(b) != "ok" {
		t.Errorf("healthz body = %q", b)
	}
}

func TestEnvOr(t *testing.T) {
	t.Setenv("BRIDGE_TEST_KEY", "set")
	if got := envOr("BRIDGE_TEST_KEY", "fallback"); got != "set" {
		t.Errorf("envOr set = %q", got)
	}
	if got := envOr("BRIDGE_TEST_UNSET_KEY", "fallback"); got != "fallback" {
		t.Errorf("envOr unset = %q", got)
	}
}

func TestServe_ConfigError(t *testing.T) {
	// Empty syslog addr fails validation before any listener is opened.
	if err := serve(context.Background(), config{network: "tcp", facility: "local0", severity: "info"}); err == nil {
		t.Fatal("want error for missing syslog addr")
	}
}

func TestServe_ShutsDownOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config{
		listen: "127.0.0.1:0", syslogAddr: "127.0.0.1:9", network: "tcp",
		appName: "x", facility: "local0", severity: "info", dialTimeout: time.Second,
	}
	done := make(chan error, 1)
	go func() { done <- serve(ctx, cfg) }()

	// Give ListenAndServe a moment to bind, then trigger graceful shutdown.
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v, want nil on clean shutdown", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not return after context cancel")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	for _, k := range []string{
		"LISTEN_ADDR", "SYSLOG_ADDR", "SYSLOG_NETWORK", "SYSLOG_TLS",
		"APP_NAME", "FACILITY", "SEVERITY", "SYSLOG_HOSTNAME", "AUTH_HEADER",
	} {
		// t.Setenv registers restoration of the prior value; immediately
		// unsetting then exercises the "env absent → fallback" path.
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("unset %s: %v", k, err)
		}
	}
	cfg := loadConfig()
	if cfg.listen != ":8514" || cfg.network != "tcp" || cfg.appName != "gameplane-audit" ||
		cfg.facility != "local0" || cfg.severity != "info" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

func TestFallbackHostname(t *testing.T) {
	cases := []struct {
		name   string
		lookup func() (string, error)
		want   string
	}{
		{"valid OS hostname kept", func() (string, error) { return "node-1", nil }, "node-1"},
		{"hostname with space dropped", func() (string, error) { return "my host", nil }, ""},
		{"non-ASCII hostname dropped", func() (string, error) { return "hôte", nil }, ""},
		{"too long hostname dropped", func() (string, error) { return strings.Repeat("h", 256), nil }, ""},
		{"lookup error dropped", func() (string, error) { return "", errors.New("no hostname") }, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fallbackHostname(tc.lookup); got != tc.want {
				t.Errorf("fallbackHostname = %q, want %q", got, tc.want)
			}
		})
	}
}

// A record sent after the collector has closed the reused connection is
// delivered on a fresh connection, not written to the closed one.
func TestForwarder_RecordAfterCollectorCloseReachesNewConnection(t *testing.T) {
	lc := &net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	firstClosed := make(chan struct{})
	second := make(chan string, 1)
	go func() {
		// Connection 1: read one frame, then close it, as a collector that
		// reaps idle connections does.
		c1, err := ln.Accept()
		if err != nil {
			return
		}
		buf := make([]byte, 256)
		_, _ = c1.Read(buf)
		_ = c1.Close()
		close(firstClosed)

		// Connection 2: accumulate everything read until "two" arrives.
		c2, err := ln.Accept()
		if err != nil {
			return
		}
		defer c2.Close()
		var acc []byte
		for {
			n, err := c2.Read(buf)
			acc = append(acc, buf[:n]...)
			if strings.Contains(string(acc), "two") {
				second <- string(acc)
				return
			}
			if err != nil {
				return
			}
		}
	}()

	f := newForwarder("tcp", ln.Addr().String(), false, time.Second)
	if err := f.send(context.Background(), []byte("one")); err != nil {
		t.Fatalf("send one: %v", err)
	}
	select {
	case <-firstClosed:
	case <-time.After(2 * time.Second):
		t.Fatal("collector did not receive the first frame")
	}
	// Wait for the collector's close to reach the forwarder's socket.
	if !waitForwarderSeesClose(t, f) {
		t.Fatal("collector close did not reach the forwarder within 2s")
	}

	if err := f.send(context.Background(), []byte("two")); err != nil {
		t.Fatalf("send two: %v", err)
	}
	select {
	case got := <-second:
		if !strings.Contains(got, "two") {
			t.Errorf("new connection received %q, want the second frame", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the record sent after the collector closed its connection did not reach a new connection")
	}
}

// The same reconnect-after-close path exercises the TLS dial branch, not just
// plain TCP: a record sent after the collector closes a reused TLS connection
// still reaches a fresh TLS connection rather than being silently lost.
func TestForwarder_TLSRecordAfterCollectorCloseReachesNewConnection(t *testing.T) {
	cert := generateTestTLSCert(t)
	pool := x509.NewCertPool()
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse leaf certificate: %v", err)
	}
	pool.AddCert(leaf)

	lc := &net.ListenConfig{}
	rawLn, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ln := tls.NewListener(rawLn, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
	defer ln.Close()

	firstClosed := make(chan struct{})
	second := make(chan string, 1)
	go func() {
		// Connection 1: read one frame, then close it, as a collector that
		// reaps idle connections does.
		c1, err := ln.Accept()
		if err != nil {
			return
		}
		buf := make([]byte, 256)
		_, _ = c1.Read(buf)
		_ = c1.Close()
		close(firstClosed)

		// Connection 2: accumulate everything read until "two" arrives.
		c2, err := ln.Accept()
		if err != nil {
			return
		}
		defer c2.Close()
		var acc []byte
		for {
			n, err := c2.Read(buf)
			acc = append(acc, buf[:n]...)
			if strings.Contains(string(acc), "two") {
				second <- string(acc)
				return
			}
			if err != nil {
				return
			}
		}
	}()

	f := newForwarder("tcp", rawLn.Addr().String(), true, time.Second)
	f.tlsConfig = &tls.Config{RootCAs: pool, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12}
	if err := f.send(context.Background(), []byte("one")); err != nil {
		t.Fatalf("send one: %v", err)
	}
	select {
	case <-firstClosed:
	case <-time.After(2 * time.Second):
		t.Fatal("collector did not receive the first frame")
	}
	// Wait for the collector's close to reach the forwarder's socket.
	if !waitForwarderSeesClose(t, f) {
		t.Fatal("collector close did not reach the forwarder within 2s")
	}

	if err := f.send(context.Background(), []byte("two")); err != nil {
		t.Fatalf("send two: %v", err)
	}
	select {
	case got := <-second:
		if !strings.Contains(got, "two") {
			t.Errorf("new connection received %q, want the second frame", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the record sent after the collector closed its TLS connection did not reach a new connection")
	}
}

// When the collector has closed the connection and is no longer reachable, the
// next POST is answered with 502 rather than reported as delivered.
func TestHandle_CollectorGoneAfterCloseIs502(t *testing.T) {
	lc := &net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	gone := make(chan struct{})
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		buf := make([]byte, 4096)
		_, _ = c.Read(buf)
		// The collector goes away: its connection and its listener both close.
		_ = c.Close()
		_ = ln.Close()
		close(gone)
	}()

	s, err := newServer(config{
		syslogAddr: ln.Addr().String(), network: "tcp", appName: "gp",
		facility: "local0", severity: "info", hostname: "h", dialTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	post := func() int {
		rr := httptest.NewRecorder()
		s.handle(rr, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(`{"k":"v"}`)))
		return rr.Code
	}

	if code := post(); code != http.StatusNoContent {
		t.Fatalf("first POST status = %d, want 204", code)
	}
	select {
	case <-gone:
	case <-time.After(2 * time.Second):
		t.Fatal("collector did not receive the first record")
	}
	// Wait for the collector's close to reach the forwarder's socket.
	if !waitForwarderSeesClose(t, s.fwd) {
		t.Fatal("collector close did not reach the forwarder within 2s")
	}

	if code := post(); code != http.StatusBadGateway {
		t.Fatalf("POST after the collector went away: status = %d, want 502", code)
	}
}

// A request whose body does not arrive within the read bound is closed by the
// server instead of holding the connection open.
func TestIntake_BodyReadIsTimeBounded(t *testing.T) {
	s := &server{network: "tcp", fwd: newForwarder("tcp", "127.0.0.1:1", false, time.Second)}

	lc := &net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := newHTTPServer(config{readTimeout: 300 * time.Millisecond}, s.routes())
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	d := &net.Dialer{Timeout: time.Second}
	conn, err := d.DialContext(context.Background(), "tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	start := time.Now()
	// Complete headers announcing a body, then no body at all.
	if _, err := io.WriteString(conn,
		"POST / HTTP/1.1\r\nHost: bridge\r\nContent-Type: application/json\r\nContent-Length: 64\r\n\r\n"); err != nil {
		t.Fatalf("write headers: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.ReadAll(conn); err != nil {
		t.Fatalf("server did not close the connection within the read bound: %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("connection closed after %v, want within 1s of the 300ms read timeout", elapsed)
	}
}

// TestForwarder_ManyHealthyReusedConnSends verifies that repeated sends over
// a single healthy reused connection complete quickly (verifying the fast path
// eliminates the 5ms penalty). With the old timed-read-only alive(), 1000 sends
// would need at least 5 seconds; with the fast path, it should complete in a
// fraction of a second.
func TestForwarder_ManyHealthyReusedConnSends(t *testing.T) {
	lc := &net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	// Collector that drains all sends without closing.
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4096)
				for {
					if _, err := c.Read(buf); err != nil {
						return
					}
				}
			}(c)
		}
	}()

	f := newForwarder("tcp", ln.Addr().String(), false, time.Second)
	start := time.Now()
	for i := 0; i < 1000; i++ {
		if err := f.send(context.Background(), []byte("x")); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	// With the old code (5ms per send * 1000), this would be >= 5s.
	// With the fast path, it should be well under 1s.
	if elapsed > 2*time.Second {
		t.Errorf("1000 sends took %v, expected < 2s (old code would need >= 5s)", elapsed)
	}
}

// The intake server bounds the header phase, the whole request and keep-alive
// idle time, and a zero read bound falls back to the default.
func TestNewHTTPServer_BoundsEveryPhase(t *testing.T) {
	if got := loadConfig().readTimeout; got != defaultReadTimeout {
		t.Errorf("default readTimeout = %v, want %v", got, defaultReadTimeout)
	}
	srv := newHTTPServer(config{listen: "127.0.0.1:0"}, http.NewServeMux())
	if srv.ReadTimeout != defaultReadTimeout {
		t.Errorf("ReadTimeout = %v, want %v", srv.ReadTimeout, defaultReadTimeout)
	}
	if srv.ReadHeaderTimeout != headerReadTimeout {
		t.Errorf("ReadHeaderTimeout = %v, want %v", srv.ReadHeaderTimeout, headerReadTimeout)
	}
	if srv.IdleTimeout != idleTimeout {
		t.Errorf("IdleTimeout = %v, want %v", srv.IdleTimeout, idleTimeout)
	}
	short := newHTTPServer(config{readTimeout: time.Second}, http.NewServeMux())
	if short.ReadHeaderTimeout != time.Second || short.ReadTimeout != time.Second {
		t.Errorf("short bound: header=%v read=%v, want 1s for both", short.ReadHeaderTimeout, short.ReadTimeout)
	}
}
