package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ValgulNecron/gameplane/api/internal/auth"
	"github.com/ValgulNecron/gameplane/api/internal/db"
	"github.com/ValgulNecron/gameplane/api/internal/httperr"
	"github.com/ValgulNecron/gameplane/api/internal/telemetry"
)

// Notice actions accepted by POST /admin/telemetry/notice.
const (
	noticeSeen        = "seen"
	noticeKeep        = "keep"
	noticeExtendedOff = "extended-off"
	noticeAllOff      = "all-off"
)

// noticeBasicFields and noticeExtendedFields are the exact report fields the
// first-login notice lists (contracts/report-schema.md).
var (
	noticeBasicFields    = []string{"version", "servers", "templates"}
	noticeExtendedFields = []string{"installId", "env.k8s", "env.distro", "env.arch", "env.nodes", "games", "features"}
)

// TelemetrySettings is the install-time telemetry setting the config and
// notice handlers need: where reports go and how far apart they are.
type TelemetrySettings struct {
	// Dest is the destination the API resolved at startup.
	Dest telemetry.Destination
	// Interval is the spacing between reports; zero means 24h.
	Interval time.Duration
}

// MountTelemetry exposes the first-login telemetry notice at
// /admin/telemetry/notice (spec 022, contracts/api-telemetry-http.md).
func MountTelemetry(r chi.Router, store *db.Store, settings TelemetrySettings) {
	h := &telemetryHandler{db: store, settings: settings}
	r.Route("/admin/telemetry", func(r chi.Router) {
		r.Get("/notice", h.getNotice)
		r.Post("/notice", h.postNotice)
	})
}

type telemetryHandler struct {
	db       *db.Store
	settings TelemetrySettings
}

// noticeDestination is the destination as shown to an admin: its kind and
// the URL host only, never the path, query or credentials.
type noticeDestination struct {
	Kind string `json:"kind"`
	Host string `json:"host"`
}

type noticeFields struct {
	Basic    []string `json:"basic"`
	Extended []string `json:"extended"`
}

// noticeResponse omits everything but pending while the notice isn't pending.
type noticeResponse struct {
	Pending     bool               `json:"pending"`
	Destination *noticeDestination `json:"destination,omitempty"`
	Fields      *noticeFields      `json:"fields,omitempty"`
}

// pending reports whether the notice is pending for u: u holds config:manage,
// a destination is in effect, no admin has made a choice yet (consent_source
// is "default"), and u has not dismissed it.
func (h *telemetryHandler) pending(ctx context.Context, u *auth.User) (bool, error) {
	if !u.Can("config:manage", false, "", "") {
		return false, nil
	}
	if h.settings.Dest.Kind == telemetry.KindDisabled || h.settings.Dest.Kind == telemetry.KindNone {
		return false, nil
	}
	st, err := h.db.GetTelemetryState(ctx)
	if err != nil {
		return false, err
	}
	if st.ConsentSource != db.TelemetryConsentDefault {
		return false, nil
	}
	acked, err := h.db.HasNoticeAck(ctx, u.ID)
	if err != nil {
		return false, err
	}
	return !acked, nil
}

func (h *telemetryHandler) getNotice(w http.ResponseWriter, req *http.Request) {
	u := auth.UserFromContext(req.Context())
	pending, err := h.pending(req.Context(), u)
	if err != nil {
		httperr.Write(w, req, err)
		return
	}
	if !pending {
		writeJSON(w, noticeResponse{})
		return
	}
	writeJSON(w, noticeResponse{
		Pending:     true,
		Destination: &noticeDestination{Kind: h.settings.Dest.Kind, Host: h.settings.Dest.Host},
		Fields:      &noticeFields{Basic: noticeBasicFields, Extended: noticeExtendedFields},
	})
}

func (h *telemetryHandler) postNotice(w http.ResponseWriter, req *http.Request) {
	u := auth.UserFromContext(req.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 1<<10)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	switch body.Action {
	case noticeSeen, noticeKeep, noticeExtendedOff, noticeAllOff:
	default:
		http.Error(w, "action must be one of seen|keep|extended-off|all-off", http.StatusBadRequest)
		return
	}
	pending, err := h.pending(req.Context(), u)
	if err != nil {
		httperr.Write(w, req, err)
		return
	}
	if !pending {
		if body.Action == noticeSeen {
			w.WriteHeader(http.StatusNoContent) // a no-op, not an error
			return
		}
		http.Error(w, "telemetry notice is not pending", http.StatusConflict)
		return
	}
	err = inTx(req.Context(), h.db, func(tx *sql.Tx) error {
		return h.apply(req.Context(), tx, u.ID, body.Action)
	})
	switch {
	case errors.Is(err, telemetry.ErrOperatorDisabled):
		http.Error(w, err.Error(), http.StatusConflict)
	case err != nil:
		httperr.Write(w, req, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// apply performs one notice action inside tx. seen records no ack; the other
// actions record the caller's ack in the same transaction as their effect.
func (h *telemetryHandler) apply(ctx context.Context, tx *sql.Tx, userID int64, action string) error {
	s := h.settings
	switch action {
	case noticeSeen:
		return telemetry.MarkNoticeSeen(ctx, tx, s.Dest, s.Interval)
	case noticeExtendedOff:
		c, err := telemetry.ReadConsentTx(ctx, tx)
		if err != nil {
			return err
		}
		if err := telemetry.ApplyConsent(ctx, tx, s.Dest, s.Interval, c.Basic, false, db.TelemetryConsentAdmin); err != nil {
			return err
		}
	case noticeAllOff:
		if err := telemetry.ApplyConsent(ctx, tx, s.Dest, s.Interval, false, false, db.TelemetryConsentAdmin); err != nil {
			return err
		}
	}
	if err := db.InsertNoticeAckTx(ctx, tx, userID, action, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	return nil
}

// inTx runs fn in a transaction on store and commits when fn succeeds. A
// SQLite store has one connection, so work that must commit together has to
// share the transaction rather than reach for the store.
func inTx(ctx context.Context, store *db.Store, fn func(tx *sql.Tx) error) error {
	tx, err := store.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
