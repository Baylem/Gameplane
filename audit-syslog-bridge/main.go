// Command audit-syslog-bridge is a tiny, generic HTTP-JSON → syslog relay. It
// accepts an HTTP POST whose body is a single JSON document and re-emits that
// document as an RFC 5424 syslog message to a configured collector (TCP, TCP
// over TLS, or UDP).
//
// It exists so Gameplane's audit webhook sink (which POSTs each audit event as
// JSON) can reach a syslog/SIEM endpoint, but it is deliberately
// schema-agnostic: it forwards the received body verbatim as the syslog MSG, so
// it works for any JSON webhook source, not just audit events.
package main

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// RFC 5424 facility and severity name → numeric maps. PRI = facility*8 + severity.
var facilities = map[string]int{
	"kern": 0, "user": 1, "mail": 2, "daemon": 3, "auth": 4, "syslog": 5,
	"lpr": 6, "news": 7, "uucp": 8, "cron": 9, "authpriv": 10, "ftp": 11,
	"local0": 16, "local1": 17, "local2": 18, "local3": 19,
	"local4": 20, "local5": 21, "local6": 22, "local7": 23,
}

var severities = map[string]int{
	"emerg": 0, "alert": 1, "crit": 2, "err": 3,
	"warning": 4, "notice": 5, "info": 6, "debug": 7,
}

// Version is set at build time via -ldflags "-X main.Version=...".
var Version = "dev"

// maxBody bounds the inbound POST body. Audit events are small; this just stops
// a misdirected client from streaming something huge at us.
const maxBody = 64 << 10

// RFC 5424 §6.2.3/§6.2.4 field-length limits for APP-NAME and HOSTNAME.
const (
	maxAppNameLen  = 48
	maxHostnameLen = 255
)

// validateRFC5424Field rejects a value that can't legally sit in an RFC 5424
// APP-NAME or HOSTNAME field: both are 1*maxLen PRINTUSASCII (%d33-126), a range
// that excludes space. A value containing a space or other non-printable
// character shifts every field after it once a collector splits the header on
// whitespace; an empty value is left alone here since it resolves to "-".
func validateRFC5424Field(name, value string, maxLen int) error {
	if value == "" {
		return nil
	}
	if len(value) > maxLen {
		return fmt.Errorf("%s %q is %d bytes, want at most %d", name, value, len(value), maxLen)
	}
	for _, r := range value {
		if r < 33 || r > 126 {
			return fmt.Errorf("%s %q contains %q, want only RFC 5424 PRINTUSASCII (33-126)", name, value, r)
		}
	}
	return nil
}

// Intake time bounds. defaultReadTimeout caps how long one request (headers and
// body) may take to arrive; headerReadTimeout caps the header phase on its own;
// idleTimeout closes keep-alive connections that sit unused between requests.
// idleTimeout is longer than Go's default client idle timeout (90s), so a Go
// sender such as the API retires an idle connection before the bridge does.
const (
	defaultReadTimeout = 15 * time.Second
	headerReadTimeout  = 10 * time.Second
	idleTimeout        = 120 * time.Second
)

// peekState values returned by peek().
type peekState int

const (
	peekUnknown peekState = iota
	peekEmpty
	peekClosed
	peekPending
)

// livenessProbe is how long send waits for a pending close from the collector
// before it reuses a stream connection. Collectors never write to a syslog
// stream, so a read that times out means the connection is still open.
const livenessProbe = 5 * time.Millisecond

type config struct {
	listen      string
	syslogAddr  string
	network     string // tcp | udp
	useTLS      bool
	appName     string
	facility    string
	severity    string
	hostname    string
	authHeader  string
	dialTimeout time.Duration
	readTimeout time.Duration
}

func loadConfig() config {
	return config{
		listen:      envOr("LISTEN_ADDR", ":8514"),
		syslogAddr:  envOr("SYSLOG_ADDR", ""),
		network:     envOr("SYSLOG_NETWORK", "tcp"),
		useTLS:      envOr("SYSLOG_TLS", "") == "true",
		appName:     envOr("APP_NAME", "gameplane-audit"),
		facility:    envOr("FACILITY", "local0"),
		severity:    envOr("SEVERITY", "info"),
		hostname:    envOr("SYSLOG_HOSTNAME", ""),
		authHeader:  envOr("AUTH_HEADER", ""),
		dialTimeout: 5 * time.Second,
		readTimeout: defaultReadTimeout,
	}
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// fallbackHostname returns the OS hostname for the RFC 5424 HOSTNAME field
// when SYSLOG_HOSTNAME is unset. A lookup error or a hostname that fails
// validateRFC5424Field yields "" instead, which the formatter renders as the
// nil value "-", so an unusual OS hostname can't shift the header fields.
func fallbackHostname(lookup func() (string, error)) string {
	h, err := lookup()
	if err != nil || validateRFC5424Field("hostname", h, maxHostnameLen) != nil {
		return ""
	}
	return h
}

// server holds the resolved relay configuration and the syslog forwarder.
type server struct {
	pri        int
	appName    string
	hostname   string
	network    string
	authHeader string
	fwd        *forwarder
}

func newServer(cfg config) (*server, error) {
	if cfg.syslogAddr == "" {
		return nil, errors.New("SYSLOG_ADDR (-syslog-addr) is required")
	}
	if cfg.network != "tcp" && cfg.network != "udp" {
		return nil, fmt.Errorf("SYSLOG_NETWORK must be tcp or udp, got %q", cfg.network)
	}
	if cfg.useTLS && cfg.network != "tcp" {
		return nil, errors.New("SYSLOG_TLS requires SYSLOG_NETWORK=tcp")
	}
	fac, ok := facilities[cfg.facility]
	if !ok {
		return nil, fmt.Errorf("unknown FACILITY %q", cfg.facility)
	}
	sev, ok := severities[cfg.severity]
	if !ok {
		return nil, fmt.Errorf("unknown SEVERITY %q", cfg.severity)
	}
	if err := validateRFC5424Field("APP_NAME", cfg.appName, maxAppNameLen); err != nil {
		return nil, err
	}
	if err := validateRFC5424Field("SYSLOG_HOSTNAME", cfg.hostname, maxHostnameLen); err != nil {
		return nil, err
	}
	host := cfg.hostname
	if host == "" {
		host = fallbackHostname(os.Hostname)
	}
	return &server{
		pri:        fac*8 + sev,
		appName:    cfg.appName,
		hostname:   host,
		network:    cfg.network,
		authHeader: cfg.authHeader,
		fwd:        newForwarder(cfg.network, cfg.syslogAddr, cfg.useTLS, cfg.dialTimeout),
	}, nil
}

func (s *server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", s.handle)
	return mux
}

func (s *server) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.authHeader != "" &&
		subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte(s.authHeader)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil {
		// Log the cause: without this an operator seeing the API's audit-webhook
		// counter tick "failed" has nothing on the bridge side to explain the gap.
		slog.Warn("read request body", "err", err)
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	line := buildSyslog(s.pri, time.Now(), s.hostname, s.appName, msg)
	if err := s.fwd.send(r.Context(), frameFor(s.network, line)); err != nil {
		slog.Error("forward to syslog failed", "err", err)
		http.Error(w, "syslog forward failed", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// buildSyslog renders one RFC 5424 message:
//
//	<PRI>1 TIMESTAMP HOSTNAME APP-NAME PROCID MSGID STRUCTURED-DATA MSG
//
// PROCID, MSGID and STRUCTURED-DATA are nil ("-"); MSG carries the JSON body
// collapsed to a single line so it stays one syslog record.
func buildSyslog(pri int, ts time.Time, host, app, msg string) string {
	if host == "" {
		host = "-"
	}
	if app == "" {
		app = "-"
	}
	msg = strings.NewReplacer("\n", " ", "\r", " ").Replace(msg)
	return fmt.Sprintf("<%d>1 %s %s %s - - - %s",
		pri, ts.UTC().Format("2006-01-02T15:04:05.000Z07:00"), host, app, msg)
}

// frameFor wraps a syslog message for the wire. TCP uses RFC 6587
// octet-counting ("<len> <msg>") so a stream collector can split records
// unambiguously; UDP sends the bare message as one datagram.
func frameFor(network, msg string) []byte {
	if network == "tcp" {
		return []byte(strconv.Itoa(len(msg)) + " " + msg)
	}
	return []byte(msg)
}

// forwarder ships framed syslog messages over a lazily-dialed, reused
// connection. Writes are serialized; a failed write triggers one reconnect and
// retry before surfacing the error.
type forwarder struct {
	network     string
	addr        string
	useTLS      bool
	dialTimeout time.Duration

	// tlsConfig overrides the default client TLS config when set. Production
	// code leaves it nil; tests use it to trust a self-signed test collector
	// certificate instead of the system root pool.
	tlsConfig *tls.Config

	mu   sync.Mutex
	conn net.Conn
}

func newForwarder(network, addr string, useTLS bool, dialTimeout time.Duration) *forwarder {
	return &forwarder{network: network, addr: addr, useTLS: useTLS, dialTimeout: dialTimeout}
}

func (f *forwarder) dial(ctx context.Context) error {
	var (
		c   net.Conn
		err error
	)
	if f.useTLS && f.network == "tcp" {
		cfg := f.tlsConfig
		if cfg == nil {
			cfg = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		d := &tls.Dialer{NetDialer: &net.Dialer{Timeout: f.dialTimeout}, Config: cfg}
		c, err = d.DialContext(ctx, "tcp", f.addr)
	} else {
		d := &net.Dialer{Timeout: f.dialTimeout}
		c, err = d.DialContext(ctx, f.network, f.addr)
	}
	if err != nil {
		return fmt.Errorf("dial %s/%s: %w", f.network, f.addr, err)
	}
	f.conn = c
	return nil
}

func (f *forwarder) send(ctx context.Context, frame []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.conn != nil && !f.alive() {
		slog.Info("syslog collector closed the connection; reconnecting",
			"network", f.network, "addr", f.addr)
		_ = f.conn.Close()
		f.conn = nil
	}
	if f.conn == nil {
		if err := f.dial(ctx); err != nil {
			return err
		}
	}
	if err := f.write(frame); err != nil {
		// The collector may have dropped a long-idle connection; reconnect once.
		_ = f.conn.Close()
		f.conn = nil
		if err := f.dial(ctx); err != nil {
			return err
		}
		if err := f.write(frame); err != nil {
			_ = f.conn.Close()
			f.conn = nil
			return fmt.Errorf("write syslog %s/%s: %w", f.network, f.addr, err)
		}
	}
	return nil
}

// alive reports whether a reused stream connection is still open. When a
// collector closes an idle connection, its FIN (or RST) is already queued on
// the socket, and a write would still succeed locally while the frame is lost.
// A short read surfaces that close first, so send moves the frame to a fresh
// connection. Datagram connections have no such state and are always reused.
//
// The probe is best-effort, not a delivery guarantee: a close that has not
// reached the socket within livenessProbe goes unseen, and the next frame can
// be lost while send reports success (204). Syslog framing has no ack, so this
// window cannot be closed without a different transport protocol.
//
// On Linux, alive first attempts a zero-wait peek to avoid the 5ms blocking
// read on every healthy reused connection; only if the peek is inconclusive
// does it fall back to the timed read.
//
// Pending data (peekPending) is deliberately not treated as proof the
// connection is open: on TLS a collector close first queues a close_notify
// alert record ahead of the FIN, so the raw socket looks readable. Falling
// through to the timed read on f.conn lets crypto/tls consume the alert and
// return io.EOF, which reports the connection as closed.
func (f *forwarder) alive() bool {
	if f.network != "tcp" {
		return true
	}

	// Fast path on Linux: non-blocking peek avoids the 5ms blocking read.
	switch peek(f.conn) {
	case peekEmpty:
		return true
	case peekClosed:
		return false
	}

	// Inconclusive or non-Linux: fall back to timed read.
	var one [1]byte
	_ = f.conn.SetReadDeadline(time.Now().Add(livenessProbe))
	_, err := f.conn.Read(one[:])
	_ = f.conn.SetReadDeadline(time.Time{})
	if err == nil {
		// The collector sent something unexpected; the connection is open.
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// write bounds a single frame write with a deadline. Without it, a collector
// that accepts the TCP connection but stops draining it would block Write
// forever while holding f.mu, wedging every subsequent audit POST behind the
// mutex until the pod is restarted.
func (f *forwarder) write(frame []byte) error {
	_ = f.conn.SetWriteDeadline(time.Now().Add(f.dialTimeout))
	_, err := f.conn.Write(frame)
	return err
}

// run wires OS signal handling to serve. main's thin wrapper; the testable
// lifecycle lives in serve.
func run(cfg config) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return serve(ctx, cfg)
}

// serve builds the relay and runs the HTTP server until ctx is cancelled (then
// it drains with a 5s deadline) or ListenAndServe fails.
func serve(ctx context.Context, cfg config) error {
	s, err := newServer(cfg)
	if err != nil {
		return err
	}
	srv := newHTTPServer(cfg, s.routes())
	slog.Info("audit-syslog-bridge listening",
		"version", Version, "listen", cfg.listen, "syslog", cfg.syslogAddr,
		"network", cfg.network, "tls", cfg.useTLS, "app", cfg.appName)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutCtx)
	}
}

// newHTTPServer builds the intake server with every phase of a request bounded
// in time: headers, the whole request (headers plus body), and keep-alive idle.
// A zero cfg.readTimeout falls back to defaultReadTimeout.
func newHTTPServer(cfg config, h http.Handler) *http.Server {
	readTimeout := cfg.readTimeout
	if readTimeout <= 0 {
		readTimeout = defaultReadTimeout
	}
	return &http.Server{
		Addr:              cfg.listen,
		Handler:           h,
		ReadHeaderTimeout: min(headerReadTimeout, readTimeout),
		ReadTimeout:       readTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func main() {
	if err := run(loadConfig()); err != nil {
		slog.Error("audit-syslog-bridge exited", "err", err)
		os.Exit(1)
	}
}
