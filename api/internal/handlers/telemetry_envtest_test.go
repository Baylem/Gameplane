//go:build envtest

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ValgulNecron/gameplane/api/internal/auth"
	"github.com/ValgulNecron/gameplane/api/internal/db"
	"github.com/ValgulNecron/gameplane/api/internal/kube"
	"github.com/ValgulNecron/gameplane/api/internal/rbac"
	"github.com/ValgulNecron/gameplane/api/internal/telemetry"
)

// Telemetry notice and config-hook tests (spec 022, US1).
//
// Each test builds its own router over its own in-memory store, with
// rbac.Middleware in front, because the notice and the config hook change
// the telemetry_state singleton: sharing captureAuditStore would let tests
// disturb each other. Callers are injected with auth.WithUser through
// doConfigAsUser (config_envtest_test.go).

var telemetryBundledDest = telemetry.Destination{
	Kind: telemetry.KindBundled, URL: "http://receiver.test:8080/ingest", Host: "receiver.test:8080",
}

// newTelemetryRouter returns a router with the config and telemetry routes
// mounted for dest, and the fresh-install store behind it (consent_source
// "default", basic and extended on, notice not shown).
func newTelemetryRouter(t *testing.T, dest telemetry.Destination) (*chi.Mux, *db.Store) {
	t.Helper()
	store, err := db.Open(context.Background(), "sqlite",
		"file:"+uniqueResourceName("telemetry")+"?mode=memory&cache=shared&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	settings := TelemetrySettings{Dest: dest, Interval: time.Hour}
	r := chi.NewRouter()
	r.Use(rbac.Middleware(kube.NewRegistry("default")))
	MountConfigWithTelemetry(r, store, nil, false, "", nil, settings)
	MountTelemetry(r, store, settings)
	return r, store
}

// telemetryAdmin is an admin ("*") with the given user ID.
func telemetryAdmin(id int64) *auth.User {
	u := configAdminUser()
	u.ID = id
	return u
}

// telemetryReader holds config:read but not config:manage.
func telemetryReader() *auth.User {
	return &auth.User{
		ID:       200,
		Username: "telemetry-reader",
		Perms: map[string]map[string]map[string]struct{}{
			"*": {"*": {"config:read": {}}},
		},
	}
}

func telemetryState(t *testing.T, store *db.Store) db.TelemetryState {
	t.Helper()
	st, err := store.GetTelemetryState(context.Background())
	if err != nil {
		t.Fatalf("get telemetry state: %v", err)
	}
	return st
}

func telemetryConfigValue(t *testing.T, store *db.Store) string {
	t.Helper()
	v, ok, err := store.ConfigValue(context.Background(), "telemetry")
	if err != nil || !ok {
		t.Fatalf("read telemetry config: %q, %v, %v", v, ok, err)
	}
	return v
}

// getNotice calls GET /admin/telemetry/notice as u and decodes the object.
func getNotice(t *testing.T, r http.Handler, u *auth.User) map[string]any {
	t.Helper()
	rr := doConfigAsUser(t, r, u, http.MethodGet, "/admin/telemetry/notice", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET notice = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode notice %q: %v", rr.Body.String(), err)
	}
	return out
}

func postNotice(t *testing.T, r http.Handler, u *auth.User, action string) *httptest.ResponseRecorder {
	t.Helper()
	return doConfigAsUser(t, r, u, http.MethodPost, "/admin/telemetry/notice", map[string]string{"action": action})
}

func TestTelemetryNotice_PendingOnlyForManagersOnADefaultInstallWithADestination(t *testing.T) {
	t.Run("a manager on a default install sees the notice", func(t *testing.T) {
		r, _ := newTelemetryRouter(t, telemetryBundledDest)
		got := getNotice(t, r, telemetryAdmin(1))
		if got["pending"] != true {
			t.Fatalf("notice = %v, want pending", got)
		}
		dest, _ := got["destination"].(map[string]any)
		if dest["kind"] != "bundled" || dest["host"] != "receiver.test:8080" {
			t.Fatalf("destination = %v, want kind bundled and the URL host only", dest)
		}
		fields, _ := got["fields"].(map[string]any)
		basic, _ := fields["basic"].([]any)
		extended, _ := fields["extended"].([]any)
		if len(basic) != 3 || len(extended) != 7 {
			t.Fatalf("fields = %v, want 3 basic and 7 extended", fields)
		}
	})

	t.Run("a reader without config:manage gets pending false only", func(t *testing.T) {
		r, _ := newTelemetryRouter(t, telemetryBundledDest)
		got := getNotice(t, r, telemetryReader())
		if got["pending"] != false || len(got) != 1 {
			t.Fatalf("notice = %v, want exactly {pending:false}", got)
		}
	})

	t.Run("a user without config:read is refused by RBAC", func(t *testing.T) {
		r, _ := newTelemetryRouter(t, telemetryBundledDest)
		rr := doConfigAsUser(t, r, configOperatorUser(), http.MethodGet, "/admin/telemetry/notice", nil)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("operator GET notice = %d, want 403", rr.Code)
		}
	})

	for _, dest := range []telemetry.Destination{{Kind: telemetry.KindNone}, {Kind: telemetry.KindDisabled}} {
		t.Run("not pending with destination "+dest.Kind, func(t *testing.T) {
			r, _ := newTelemetryRouter(t, dest)
			if got := getNotice(t, r, telemetryAdmin(1)); got["pending"] != false || len(got) != 1 {
				t.Fatalf("notice = %v, want exactly {pending:false}", got)
			}
		})
	}

	t.Run("not pending once an admin made a choice", func(t *testing.T) {
		r, store := newTelemetryRouter(t, telemetryBundledDest)
		rr := doConfigAsUser(t, r, telemetryAdmin(1), http.MethodPut, "/admin/config/telemetry",
			map[string]bool{"sendMetrics": true, "extended": true})
		if rr.Code != http.StatusOK {
			t.Fatalf("PUT = %d; body=%s", rr.Code, rr.Body.String())
		}
		if got := getNotice(t, r, telemetryAdmin(2)); got["pending"] != false {
			t.Fatalf("notice = %v, want not pending after an explicit choice", got)
		}
		if st := telemetryState(t, store); st.ConsentSource != db.TelemetryConsentAdmin {
			t.Fatalf("consent source = %q, want admin", st.ConsentSource)
		}
	})
}

func TestTelemetryNotice_SeenSetsNoticeShownOnce(t *testing.T) {
	r, store := newTelemetryRouter(t, telemetryBundledDest)
	admin := telemetryAdmin(1)

	if st := telemetryState(t, store); st.NoticeShownAt != "" || st.NextDueAt != "" {
		t.Fatalf("fresh state = %+v, want no notice and no slot", st)
	}
	if rr := postNotice(t, r, admin, "seen"); rr.Code != http.StatusNoContent {
		t.Fatalf("seen = %d, want 204; body=%s", rr.Code, rr.Body.String())
	}
	first := telemetryState(t, store)
	if first.NoticeShownAt == "" || first.NextDueAt == "" || first.InstallID == "" {
		t.Fatalf("state after seen = %+v, want the notice stamped, the schedule open and an install id", first)
	}
	if first.ConsentSource != db.TelemetryConsentDefault {
		t.Fatalf("consent source = %q, seen must leave it as default", first.ConsentSource)
	}
	if acked, err := store.HasNoticeAck(context.Background(), admin.ID); err != nil || acked {
		t.Fatalf("HasNoticeAck = %v, %v; seen must record no ack", acked, err)
	}

	if rr := postNotice(t, r, admin, "seen"); rr.Code != http.StatusNoContent {
		t.Fatalf("second seen = %d, want 204", rr.Code)
	}
	if again := telemetryState(t, store); again != first {
		t.Fatalf("a second seen changed the state: %+v then %+v", first, again)
	}
	if got := getNotice(t, r, admin); got["pending"] != true {
		t.Fatal("seen alone must not dismiss the notice")
	}
}

func TestTelemetryNotice_SeenIsANoOpWhenNotPending(t *testing.T) {
	r, store := newTelemetryRouter(t, telemetry.Destination{Kind: telemetry.KindNone})
	if rr := postNotice(t, r, telemetryAdmin(1), "seen"); rr.Code != http.StatusNoContent {
		t.Fatalf("seen = %d, want 204", rr.Code)
	}
	if st := telemetryState(t, store); st.NoticeShownAt != "" {
		t.Fatalf("notice_shown_at = %q, want it untouched when nothing is pending", st.NoticeShownAt)
	}
}

func TestTelemetryNotice_KeepOnlyAcksTheCaller(t *testing.T) {
	r, store := newTelemetryRouter(t, telemetryBundledDest)
	admin := telemetryAdmin(1)
	before := telemetryConfigValue(t, store)

	if rr := postNotice(t, r, admin, "keep"); rr.Code != http.StatusNoContent {
		t.Fatalf("keep = %d, want 204; body=%s", rr.Code, rr.Body.String())
	}
	if got := telemetryConfigValue(t, store); got != before {
		t.Fatalf("config = %s, keep must leave consent unchanged (was %s)", got, before)
	}
	if st := telemetryState(t, store); st.ConsentSource != db.TelemetryConsentDefault {
		t.Fatalf("consent source = %q, keep must leave it as default", st.ConsentSource)
	}
	if acked, err := store.HasNoticeAck(context.Background(), admin.ID); err != nil || !acked {
		t.Fatalf("HasNoticeAck = %v, %v; want the caller's ack recorded", acked, err)
	}
	if got := getNotice(t, r, admin); got["pending"] != false {
		t.Fatalf("notice = %v, want it dismissed for the caller", got)
	}
	if got := getNotice(t, r, telemetryAdmin(2)); got["pending"] != true {
		t.Fatalf("notice = %v, want it still pending for another admin", got)
	}
}

func TestTelemetryNotice_ExtendedOffKeepsBasic(t *testing.T) {
	r, store := newTelemetryRouter(t, telemetryBundledDest)
	admin := telemetryAdmin(1)
	if rr := postNotice(t, r, admin, "seen"); rr.Code != http.StatusNoContent {
		t.Fatalf("seen = %d", rr.Code)
	}
	if telemetryState(t, store).InstallID == "" {
		t.Fatal("setup: seen should have created the install id")
	}

	if rr := postNotice(t, r, admin, "extended-off"); rr.Code != http.StatusNoContent {
		t.Fatalf("extended-off = %d, want 204; body=%s", rr.Code, rr.Body.String())
	}
	if got := telemetryConfigValue(t, store); got != `{"sendMetrics":true,"extended":false}` {
		t.Fatalf("config = %s, want basic on and extended off", got)
	}
	st := telemetryState(t, store)
	if st.ConsentSource != db.TelemetryConsentAdmin || st.InstallID != "" || st.NextDueAt == "" {
		t.Fatalf("state = %+v, want source admin, the ID deleted and basic reporting still scheduled", st)
	}
	if acked, err := store.HasNoticeAck(context.Background(), admin.ID); err != nil || !acked {
		t.Fatalf("HasNoticeAck = %v, %v; want an ack", acked, err)
	}
}

func TestTelemetryNotice_AllOffStopsEverything(t *testing.T) {
	r, store := newTelemetryRouter(t, telemetryBundledDest)
	admin := telemetryAdmin(1)
	if rr := postNotice(t, r, admin, "seen"); rr.Code != http.StatusNoContent {
		t.Fatalf("seen = %d", rr.Code)
	}

	if rr := postNotice(t, r, admin, "all-off"); rr.Code != http.StatusNoContent {
		t.Fatalf("all-off = %d, want 204; body=%s", rr.Code, rr.Body.String())
	}
	if got := telemetryConfigValue(t, store); got != `{"sendMetrics":false,"extended":false}` {
		t.Fatalf("config = %s, want both tiers off", got)
	}
	st := telemetryState(t, store)
	if st.ConsentSource != db.TelemetryConsentAdmin || st.InstallID != "" || st.NextDueAt != "" {
		t.Fatalf("state = %+v, want source admin, no ID and the schedule closed", st)
	}
	if acked, err := store.HasNoticeAck(context.Background(), admin.ID); err != nil || !acked {
		t.Fatalf("HasNoticeAck = %v, %v; want an ack", acked, err)
	}
}

func TestTelemetryNotice_ActionsConflictWhenNotPending(t *testing.T) {
	r, _ := newTelemetryRouter(t, telemetryBundledDest)
	admin := telemetryAdmin(1)
	if rr := postNotice(t, r, admin, "keep"); rr.Code != http.StatusNoContent {
		t.Fatalf("keep = %d", rr.Code)
	}
	for _, action := range []string{"keep", "extended-off", "all-off"} {
		if rr := postNotice(t, r, admin, action); rr.Code != http.StatusConflict {
			t.Errorf("%s after dismissal = %d, want 409", action, rr.Code)
		}
	}

	none, _ := newTelemetryRouter(t, telemetry.Destination{Kind: telemetry.KindNone})
	if rr := postNotice(t, none, admin, "all-off"); rr.Code != http.StatusConflict {
		t.Errorf("all-off with no destination = %d, want 409", rr.Code)
	}
}

func TestTelemetryNotice_BadRequestsAndForbidden(t *testing.T) {
	r, _ := newTelemetryRouter(t, telemetryBundledDest)
	admin := telemetryAdmin(1)

	if rr := postNotice(t, r, admin, "explode"); rr.Code != http.StatusBadRequest {
		t.Errorf("unknown action = %d, want 400", rr.Code)
	}
	req := httptest.NewRequestWithContext(auth.WithUser(context.Background(), admin),
		http.MethodPost, "/admin/telemetry/notice", strings.NewReader("{nope"))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("malformed body = %d, want 400", rr.Code)
	}
	if rr := postNotice(t, r, telemetryReader(), "keep"); rr.Code != http.StatusForbidden {
		t.Errorf("POST without config:manage = %d, want 403", rr.Code)
	}
}

func TestTelemetryConfig_BasicOffStoresExtendedOffAndDeletesID(t *testing.T) {
	r, store := newTelemetryRouter(t, telemetryBundledDest)
	admin := telemetryAdmin(1)
	put := func(sendMetrics, extended bool) *httptest.ResponseRecorder {
		return doConfigAsUser(t, r, admin, http.MethodPut, "/admin/config/telemetry",
			map[string]bool{"sendMetrics": sendMetrics, "extended": extended})
	}

	if rr := put(true, true); rr.Code != http.StatusOK {
		t.Fatalf("PUT on = %d; body=%s", rr.Code, rr.Body.String())
	}
	if telemetryState(t, store).InstallID == "" {
		t.Fatal("setup: turning extended on should create the install id")
	}

	rr := put(false, true)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT {sendMetrics:false, extended:true} = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Section string          `json:"section"`
		Value   json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Section != "telemetry" || string(resp.Value) != `{"sendMetrics":false,"extended":false}` {
		t.Fatalf("response = %s, want the normalised value with extended false", rr.Body.String())
	}
	if got := telemetryConfigValue(t, store); got != `{"sendMetrics":false,"extended":false}` {
		t.Fatalf("stored config = %s, want extended stored off", got)
	}
	st := telemetryState(t, store)
	if st.InstallID != "" || st.NextDueAt != "" || st.ConsentSource != db.TelemetryConsentAdmin {
		t.Fatalf("state = %+v, want the ID deleted, the schedule closed and source admin", st)
	}
}

func TestTelemetryConfig_SaveCreatesAndDeletesInstallIDAndSetsAdminSource(t *testing.T) {
	r, store := newTelemetryRouter(t, telemetryBundledDest)
	admin := telemetryAdmin(1)
	put := func(sendMetrics, extended bool) {
		t.Helper()
		rr := doConfigAsUser(t, r, admin, http.MethodPut, "/admin/config/telemetry",
			map[string]bool{"sendMetrics": sendMetrics, "extended": extended})
		if rr.Code != http.StatusOK {
			t.Fatalf("PUT {%v,%v} = %d; body=%s", sendMetrics, extended, rr.Code, rr.Body.String())
		}
	}

	if telemetryState(t, store).ConsentSource != db.TelemetryConsentDefault {
		t.Fatal("setup: a fresh install starts with source default")
	}
	put(true, true)
	st := telemetryState(t, store)
	if st.ConsentSource != db.TelemetryConsentAdmin || st.InstallID == "" || st.NextDueAt == "" {
		t.Fatalf("after extended on: %+v, want source admin, an ID and an open schedule", st)
	}
	put(true, false)
	if st := telemetryState(t, store); st.InstallID != "" || st.NextDueAt == "" {
		t.Fatalf("after extended off: %+v, want the ID deleted and basic still scheduled", st)
	}
	put(true, true)
	if again := telemetryState(t, store); again.InstallID == "" || again.InstallID == st.InstallID {
		t.Fatalf("after extended on again: id %q (was %q), want a new one", again.InstallID, st.InstallID)
	}
}

func TestTelemetryConfig_OperatorDisabledIsRefusedWith409(t *testing.T) {
	r, store := newTelemetryRouter(t, telemetry.Destination{Kind: telemetry.KindDisabled})
	before := telemetryConfigValue(t, store)

	rr := doConfigAsUser(t, r, telemetryAdmin(1), http.MethodPut, "/admin/config/telemetry",
		map[string]bool{"sendMetrics": false, "extended": false})
	if rr.Code != http.StatusConflict {
		t.Fatalf("PUT = %d, want 409; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "telemetry is disabled by the operator") {
		t.Fatalf("body = %q, want the operator-disabled message", rr.Body.String())
	}
	if got := telemetryConfigValue(t, store); got != before {
		t.Fatalf("config = %s, a refused save must change nothing (was %s)", got, before)
	}
	if st := telemetryState(t, store); st.ConsentSource != db.TelemetryConsentDefault {
		t.Fatalf("consent source = %q, a refused save must leave it alone", st.ConsentSource)
	}
}

func TestTelemetryConfig_PutNeedsManagePermission(t *testing.T) {
	r, _ := newTelemetryRouter(t, telemetryBundledDest)
	rr := doConfigAsUser(t, r, telemetryReader(), http.MethodPut, "/admin/config/telemetry",
		map[string]bool{"sendMetrics": false, "extended": false})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("PUT without config:manage = %d, want 403", rr.Code)
	}
}
