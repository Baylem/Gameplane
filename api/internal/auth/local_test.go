package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ValgulNecron/gameplane/api/internal/db"
)

func newAuthDB(t *testing.T) *db.Store {
	t.Helper()
	s, err := db.Open(context.Background(), "sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return s
}

func seedUser(t *testing.T, s *db.Store, username, pw, role string) {
	t.Helper()
	SetFastHashParams(t)
	hash, err := HashPassword(pw)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	_, err = s.DB.ExecContext(context.Background(),
		`INSERT INTO users(username, display_name, email, role, pw_hash) VALUES (?,?,?,?,?)`,
		username, username, username+"@example.com", role, hash,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
}

func TestLogin_RejectsNonJSONContentType(t *testing.T) {
	s := newAuthDB(t)
	l := NewLocal(s)
	ss := NewSessionStore(s)
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/login", strings.NewReader("user=alice"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	l.HandleLogin(ss, nil).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestLogin_BadJSON(t *testing.T) {
	s := newAuthDB(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/login", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	NewLocal(s).HandleLogin(NewSessionStore(s), nil).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestLogin_UnknownUser_TimingPath(t *testing.T) {
	s := newAuthDB(t)
	body := strings.NewReader(`{"username":"ghost","password":"hunter2"}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")
	NewLocal(s).HandleLogin(NewSessionStore(s), nil).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	s := newAuthDB(t)
	seedUser(t, s, "alice", "rightpw", "admin")
	body := strings.NewReader(`{"username":"alice","password":"wrong"}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")
	NewLocal(s).HandleLogin(NewSessionStore(s), nil).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestLogin_Success_SetsCookies(t *testing.T) {
	s := newAuthDB(t)
	seedUser(t, s, "alice", "hunter2", "admin")
	// Seed a cluster-wide viewer role binding so perms actually load.
	if _, err := s.DB.ExecContext(context.Background(),
		`INSERT INTO user_role_bindings(user_id, role_name, cluster, namespace) VALUES (1, 'viewer', 'local', '*')`,
	); err != nil {
		t.Fatalf("seed binding: %v", err)
	}
	body := strings.NewReader(`{"username":"alice","password":"hunter2"}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")
	NewLocal(s).HandleLogin(NewSessionStore(s), nil).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body)
	}
	// Assert raw JSON uses camelCase and not PascalCase.
	bodyBytes := rr.Body.Bytes()
	bodyStr := string(bodyBytes)
	if !strings.Contains(bodyStr, `"username"`) {
		t.Fatalf("raw JSON missing camelCase 'username': %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"permissions"`) {
		t.Fatalf("raw JSON missing camelCase 'permissions': %s", bodyStr)
	}
	if strings.Contains(bodyStr, `"Username"`) {
		t.Fatalf("raw JSON contains PascalCase 'Username': %s", bodyStr)
	}
	if strings.Contains(bodyStr, `"Perms"`) {
		t.Fatalf("raw JSON contains PascalCase 'Perms': %s", bodyStr)
	}
	// Decode and verify permissions loaded.
	var got loginResp
	if err := json.NewDecoder(strings.NewReader(bodyStr)).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.User.Username != "alice" || got.CSRF == "" {
		t.Fatalf("got %+v", got)
	}
	// Viewer role grants servers:read cluster-wide.
	if len(got.User.Permissions) == 0 || len(got.User.Permissions["*"]) == 0 {
		t.Fatalf("permissions empty or missing cluster-wide: %+v", got.User.Permissions)
	}
	var hasServersRead bool
	for _, p := range got.User.Permissions["*"] {
		if p == "servers:read" {
			hasServersRead = true
			break
		}
	}
	if !hasServersRead {
		t.Fatalf("cluster-wide permissions missing 'servers:read': %+v", got.User.Permissions["*"])
	}
	cookies := rr.Result().Cookies()
	var session, csrf bool
	for _, c := range cookies {
		if c.Name == sessionCookie {
			session = true
		}
		if c.Name == csrfCookie {
			csrf = true
		}
	}
	if !session || !csrf {
		t.Fatalf("cookies not set: %+v", cookies)
	}
}

func TestLogin_PerUserRateLimit(t *testing.T) {
	s := newAuthDB(t)
	seedUser(t, s, "alice", "hunter2", "admin")
	// Drain the per-user bucket for "alice".
	for i := 0; i < 6; i++ {
		LoginUserLimiter.AllowUser("alice")
	}
	body := strings.NewReader(`{"username":"alice","password":"hunter2"}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")
	NewLocal(s).HandleLogin(NewSessionStore(s), nil).ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("code=%d", rr.Code)
	}
}

// TestLogin_EmbedsPreferences verifies POST /auth/login's response carries
// the same "preferences" shape GET /users/me does (D3, 2026-09-23) so the
// dashboard can seed the correct theme without a second round-trip.
func TestLogin_EmbedsPreferences(t *testing.T) {
	s := newAuthDB(t)
	seedUser(t, s, "prefs-alice", "hunter2", "admin")
	// Look up the seeded user's id to seed preferences through the store API.
	var id int64
	if err := s.DB.QueryRowContext(context.Background(), "SELECT id FROM users WHERE username = ?", "prefs-alice").Scan(&id); err != nil {
		t.Fatalf("query user id: %v", err)
	}
	// Seed a non-default preferences row so the assertion can't pass on
	// DefaultUserPreferences() alone.
	if _, err := s.UpsertPreferences(context.Background(), id, db.UserPreferences{
		ThemeType:      "preset",
		PresetID:       "legacy",
		AppearanceMode: "dark",
	}); err != nil {
		t.Fatalf("seed preferences: %v", err)
	}

	body := strings.NewReader(`{"username":"prefs-alice","password":"hunter2"}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")
	NewLocal(s).HandleLogin(NewSessionStore(s), nil).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body)
	}
	if !strings.Contains(rr.Body.String(), `"preferences"`) {
		t.Fatalf("login response missing 'preferences': %s", rr.Body)
	}
	var got loginResp
	if err := json.NewDecoder(strings.NewReader(rr.Body.String())).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.User.Preferences == nil {
		t.Fatal("decoded login response has nil Preferences")
	}
	if got.User.Preferences.PresetID != "legacy" || got.User.Preferences.AppearanceMode != "dark" {
		t.Errorf("preferences = %+v, want legacy/dark", got.User.Preferences)
	}
}
