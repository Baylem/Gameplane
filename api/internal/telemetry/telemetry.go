// Package telemetry sends the install's usage report to its resolved
// destination (specs/022-default-telemetry-dashboard).
//
// Privacy: the basic report carries only the control-plane version and
// total counts. The extended report adds an install ID and environment
// categories, only while the admin's extended switch is on, and is signed.
// Nothing is sent while the operator disabled telemetry, no destination is in
// effect, the admin turned basic off, or (on a fresh install) before the
// notice was shown to an admin (FR-004).
//
// The schedule lives in the database (telemetry_state.next_due_at), so a
// restart never sends early and replicas sharing one database claim each
// slot with a conditional UPDATE (research R12).
package telemetry

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ValgulNecron/gameplane/api/internal/db"
	"github.com/ValgulNecron/gameplane/telemetryschema"
)

const (
	// maxPoll caps how often the reporter looks at the schedule.
	maxPoll = 5 * time.Minute
	// baseBackoff is the wait after the first failure; it doubles per
	// further failure up to the report interval.
	baseBackoff = time.Hour
	// extFallbackFor is how long the extended part is withheld from an
	// endpoint that answered 400 to it (research R2).
	extFallbackFor = 7 * 24 * time.Hour
	// outcomeOK and outcomeFailed are the telemetry_state.last_outcome values.
	outcomeOK     = "ok"
	outcomeFailed = "failed"
)

// Config configures a Reporter.
type Config struct {
	// Dest is the resolved destination. Kinds disabled and none never send.
	Dest Destination
	// Interval is the spacing between reports; non-positive means 24h.
	Interval time.Duration
	// Auth, when non-empty, is sent verbatim as the Authorization header.
	Auth string
	// Deps is what Collect reads from. Deps.Store is required. Extended and
	// Now are set per report by the reporter.
	Deps Deps
	// Client is the HTTP client; nil means one with a 15s timeout.
	Client *http.Client
	// Now returns the current time; nil means time.Now. Tests set it.
	Now func() time.Time
	// Rand returns a uniform value in [0, n); nil means math/rand/v2.
	Rand func(n int64) int64
}

// Reporter sends telemetry reports on the persisted schedule.
type Reporter struct {
	dest     Destination
	interval time.Duration
	auth     string
	deps     Deps
	store    *db.Store
	client   *http.Client
	now      func() time.Time
	rnd      func(int64) int64
}

// New builds a Reporter from cfg.
func New(cfg Config) *Reporter {
	if cfg.Interval <= 0 {
		cfg.Interval = 24 * time.Hour
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 15 * time.Second}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Rand == nil {
		cfg.Rand = randN
	}
	return &Reporter{
		dest:     cfg.Dest,
		interval: cfg.Interval,
		auth:     cfg.Auth,
		deps:     cfg.Deps,
		store:    cfg.Deps.Store,
		client:   cfg.Client,
		now:      cfg.Now,
		rnd:      cfg.Rand,
	}
}

// pollEvery is how often Run evaluates the schedule: min(5m, interval/4).
func (r *Reporter) pollEvery() time.Duration {
	return max(time.Second, min(maxPoll, r.interval/4))
}

// Run evaluates the schedule every pollEvery until ctx is cancelled. It
// returns at once, making no connection, when the destination is disabled or
// none.
func (r *Reporter) Run(ctx context.Context) {
	if !destActive(r.dest) {
		slog.Info("telemetry off: no destination in effect", "kind", r.dest.Kind)
		return
	}
	t := time.NewTicker(r.pollEvery())
	defer t.Stop()
	for {
		if err := r.Tick(ctx); err != nil {
			slog.Warn("telemetry report", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Tick evaluates the schedule once. It opens or closes the gate's slot,
// and, when a slot is due and this process wins the claim, sends one report.
// It returns an error when the send failed (the failure is also recorded).
func (r *Reporter) Tick(ctx context.Context) error {
	now := r.now()
	st, err := r.store.GetTelemetryState(ctx)
	if err != nil {
		return fmt.Errorf("telemetry tick: %w", err)
	}
	c, err := ReadConsent(ctx, r.store)
	if err != nil {
		return fmt.Errorf("telemetry tick: %w", err)
	}
	if !gateOpen(destActive(r.dest) && c.Basic, st) {
		if st.NextDueAt == "" {
			return nil
		}
		if err := r.store.CloseTelemetrySchedule(ctx); err != nil {
			return fmt.Errorf("telemetry tick: %w", err)
		}
		return nil
	}
	if st.NextDueAt == "" {
		if _, err := r.store.OpenTelemetrySchedule(ctx, stamp(firstDue(now, r.interval, r.rnd))); err != nil {
			return fmt.Errorf("telemetry tick: %w", err)
		}
		return nil
	}
	due, perr := time.Parse(time.RFC3339, st.NextDueAt)
	if perr != nil {
		due = now // a corrupt slot counts as due, and the claim replaces it
	}
	if now.Before(due) {
		return nil
	}
	return r.attempt(ctx, st, c, now)
}

// jitter returns U(0, d).
func (r *Reporter) jitter(d time.Duration) time.Duration {
	return time.Duration(r.rnd(int64(d)))
}

// extWithheld reports whether the extended part is withheld because this
// endpoint answered 400 to it within the last 7 days.
func (r *Reporter) extWithheld(st db.TelemetryState, now time.Time) bool {
	if st.ExtUnsupportedEndpoint != r.dest.URL {
		return false
	}
	until, err := time.Parse(time.RFC3339, st.ExtUnsupportedUntil)
	return err == nil && now.Before(until)
}

// attempt claims the due slot and, if it won, builds, signs and sends one
// report, then records the outcome.
func (r *Reporter) attempt(ctx context.Context, st db.TelemetryState, c Consent, now time.Time) error {
	claimed := stamp(now.Add(r.interval + r.jitter(r.interval/48)))
	won, err := r.store.ClaimTelemetryDue(ctx, st.NextDueAt, claimed, stamp(now))
	if err != nil {
		return fmt.Errorf("telemetry attempt: %w", err)
	}
	if !won {
		return nil // another replica sent this slot
	}
	rep, err := r.build(ctx, st, c.Extended && !r.extWithheld(st, now), now)
	if err != nil {
		return r.finish(ctx, st, claimed, err)
	}
	status, err := r.send(ctx, rep)
	if err == nil && status == http.StatusBadRequest && rep.Ext != nil {
		// The endpoint doesn't understand the extended part: re-send the
		// basic report once and stop sending ext to it for a week. A 409
		// (ID rotation) branch joins this switch in US7.
		basic := rep
		basic.Ext = nil
		status, err = r.send(ctx, basic)
		if mErr := r.store.SetExtUnsupported(ctx, stamp(now.Add(extFallbackFor)), r.dest.URL); mErr != nil {
			slog.Warn("telemetry fallback marker", "err", mErr)
		}
	}
	if err == nil && (status < 200 || status > 299) {
		err = fmt.Errorf("telemetry endpoint returned %d", status)
	}
	return r.finish(ctx, st, claimed, err)
}

// finish records the outcome of a claimed attempt: cause is nil on success.
// On success the next slot is interval plus jitter after completion, so
// reports are at least one interval apart. On failure it backs off
// min(1h*2^n, interval). cause is returned.
func (r *Reporter) finish(ctx context.Context, st db.TelemetryState, claimed string, cause error) error {
	done := r.now()
	res := db.TelemetryResult{ExpectedNext: claimed}
	if cause == nil {
		res.Outcome = outcomeOK
		res.SuccessAt = stamp(done)
		res.NextDue = stamp(done.Add(r.interval + r.jitter(r.interval/48)))
	} else {
		res.Outcome = outcomeFailed
		res.Failures = st.ConsecutiveFailures + 1
		res.NextDue = stamp(done.Add(r.backoff(st.ConsecutiveFailures)))
	}
	if err := r.store.RecordTelemetryResult(ctx, res); err != nil {
		return errors.Join(cause, fmt.Errorf("telemetry attempt: %w", err))
	}
	return cause
}

// backoff is min(1h*2^failures, interval).
func (r *Reporter) backoff(failures int) time.Duration {
	d := baseBackoff
	for i := 0; i < failures && d < r.interval; i++ {
		d *= 2
	}
	return min(d, r.interval)
}

// build collects the report. extended is the gate's extendedOn minus the
// fallback marker; a fresh install is seeded with extended on and no install
// ID yet, so the ID is created here when missing.
func (r *Reporter) build(ctx context.Context, st db.TelemetryState, extended bool, now time.Time) (telemetryschema.Report, error) {
	if extended && st.InstallID == "" {
		id, err := NewInstallID()
		if err != nil {
			return telemetryschema.Report{}, fmt.Errorf("telemetry build: %w", err)
		}
		if err := r.store.SetInstallID(ctx, id); err != nil {
			return telemetryschema.Report{}, fmt.Errorf("telemetry build: %w", err)
		}
	}
	deps := r.deps
	deps.Extended = extended
	deps.Now = func() time.Time { return now }
	rep, err := Collect(ctx, deps)
	if err != nil {
		return telemetryschema.Report{}, fmt.Errorf("telemetry build: %w", err)
	}
	return rep, nil
}

// send encodes rep and POSTs it, signing the exact body bytes when rep
// carries the extended part (the signature header is absent otherwise,
// FR-038). It returns the HTTP status.
func (r *Reporter) send(ctx context.Context, rep telemetryschema.Report) (int, error) {
	body, err := telemetryschema.Encode(rep)
	if err != nil {
		return 0, fmt.Errorf("telemetry send: %w", err)
	}
	sig := ""
	if rep.Ext != nil {
		secret, err := r.store.EnsureSigningSecret(ctx)
		if err != nil {
			return 0, fmt.Errorf("telemetry send: %w", err)
		}
		priv, err := telemetryschema.DeriveKey(secret, rep.Ext.InstallID)
		if err != nil {
			return 0, fmt.Errorf("telemetry send: %w", err)
		}
		sig = telemetryschema.Sign(priv, body)
	}
	return r.post(ctx, body, sig)
}

// post sends one request and returns the status code.
func (r *Reporter) post(ctx context.Context, body []byte, sig string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.dest.URL, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("telemetry post: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if r.auth != "" {
		req.Header.Set("Authorization", r.auth)
	}
	if sig != "" {
		req.Header.Set(telemetryschema.SignatureHeader, sig)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("telemetry post: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, nil
}
