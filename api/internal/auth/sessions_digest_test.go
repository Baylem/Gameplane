package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionDigest_Deterministic(t *testing.T) {
	a := sessionDigest("a")
	if a != sessionDigest("a") {
		t.Error("sessionDigest not deterministic")
	}
	b := sessionDigest("b")
	if a == b {
		t.Error("sessionDigest('a') == sessionDigest('b')")
	}
	if len(a) != 64 {
		t.Errorf("digest length %d, want 64", len(a))
	}
	expected := hex.EncodeToString(sha256.Sum256([]byte("a"))[:])
	if a != expected {
		t.Errorf("sessionDigest('a') = %s, want %s", a, expected)
	}
}

func TestSessions_StoresDigestNotCookieValue(t *testing.T) {
	s := newAuthDB(t)
	ctx := context.Background()
	seedUser(t, s, "alice", "pw", "admin")
	tok, _, err := s.Create(ctx, 1)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var stored string
	err = s.db.DB.QueryRowContext(ctx, `SELECT token FROM sessions WHERE user_id=1`).Scan(&stored)
	if err != nil {
		t.Fatalf("query token: %v", err)
	}
	if stored == tok {
		t.Error("stored token == raw cookie value (should be digest)")
	}
	if stored != sessionDigest(tok) {
		t.Errorf("stored token %s, want sessionDigest(%s)=%s", stored, tok, sessionDigest(tok))
	}
	var count int
	err = s.db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE token = ?`, tok).Scan(&count)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Errorf("count rows WHERE token=raw value: %d, want 0", count)
	}
}

func TestSessions_StoredValueCannotBeUsedAsCookie(t *testing.T) {
	s := newAuthDB(t)
	ctx := context.Background()
	seedUser(t, s, "alice", "pw", "admin")
	tok, _, err := s.Create(ctx, 1)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var stored string
	err = s.db.DB.QueryRowContext(ctx, `SELECT token FROM sessions WHERE user_id=1`).Scan(&stored)
	if err != nil {
		t.Fatalf("query token: %v", err)
	}

	called := false
	handler := s.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	// Try with stored digest as cookie - should fail
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: stored})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("GET with stored digest cookie: %d, want 401", rr.Code)
	}
	if called {
		t.Error("handler called with stored digest cookie")
	}

	// Try with raw token as cookie - should succeed
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: tok})
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("GET with raw token cookie: %d, want 204", rr.Code)
	}
	if !called {
		t.Error("handler not called with raw token cookie")
	}

	// lookup with stored digest should error
	_, _, err = s.lookup(ctx, stored)
	if err == nil {
		t.Error("lookup(stored digest) should error")
	}

	// POST with stored digest as cookie and correct CSRF should still fail
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: stored})
	req.Header.Set(csrfHeader, "anything") // Wrong but proves CSRF isn't checked first
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("POST with stored digest cookie: %d, want 401", rr.Code)
	}
}

func TestSessions_LogoutDeletesRowByDigest(t *testing.T) {
	s := newAuthDB(t)
	ctx := context.Background()
	seedUser(t, s, "alice", "pw", "admin")

	// Create two sessions
	tokA, _, err := s.Create(ctx, 1)
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	tokB, _, err := s.Create(ctx, 1)
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}

	// Logout with tokA
	handler := s.HandleLogout()
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: tokA})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Check tokA deleted, tokB still exists
	var countA, countB int
	s.db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE token = ?`, sessionDigest(tokA)).Scan(&countA)
	s.db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE token = ?`, sessionDigest(tokB)).Scan(&countB)

	if countA != 0 {
		t.Errorf("tokA row count after logout: %d, want 0", countA)
	}
	if countB != 1 {
		t.Errorf("tokB row count after logout: %d, want 1", countB)
	}

	// tokB should still authenticate
	_, _, err = s.lookup(ctx, tokB)
	if err != nil {
		t.Errorf("lookup(tokB) after logout: %v", err)
	}

	// Logout presenting the stored digest as cookie should delete nothing
	var stored string
	s.db.DB.QueryRowContext(ctx, `SELECT token FROM sessions WHERE user_id=1`).Scan(&stored)
	req = httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: stored})
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	var countUser int
	s.db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE user_id=1`).Scan(&countUser)
	if countUser != 1 {
		t.Errorf("user 1 row count after logout with digest: %d, want 1", countUser)
	}
}

func TestSessions_ExpiredDigestRowRemovedOnLookupAndGC(t *testing.T) {
	s := newAuthDB(t)
	ctx := context.Background()
	seedUser(t, s, "alice", "pw", "admin")

	tokA, _, err := s.Create(ctx, 1)
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	tokB, _, err := s.Create(ctx, 1)
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}

	// Set both to expired
	expiredTime := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	_, err = s.db.DB.ExecContext(ctx,
		`UPDATE sessions SET expires_at = ? WHERE token = ?`,
		expiredTime, sessionDigest(tokA))
	if err != nil {
		t.Fatalf("update tokA expiry: %v", err)
	}
	_, err = s.db.DB.ExecContext(ctx,
		`UPDATE sessions SET expires_at = ? WHERE token = ?`,
		expiredTime, sessionDigest(tokB))
	if err != nil {
		t.Fatalf("update tokB expiry: %v", err)
	}

	// lookup tokA should error and remove row
	_, _, err = s.lookup(ctx, tokA)
	if err == nil {
		t.Error("lookup(expired tokA) should error")
	}
	var countA int
	s.db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE token = ?`, sessionDigest(tokA)).Scan(&countA)
	if countA != 0 {
		t.Errorf("tokA row count after lookup: %d, want 0", countA)
	}

	// GC should remove tokB
	s.gcOnce(ctx)
	var countB int
	s.db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE token = ?`, sessionDigest(tokB)).Scan(&countB)
	if countB != 0 {
		t.Errorf("tokB row count after gc: %d, want 0", countB)
	}

	var countAll int
	s.db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions`).Scan(&countAll)
	if countAll != 0 {
		t.Errorf("total session count: %d, want 0", countAll)
	}
}

func TestSessions_DeleteForUserRevokesEveryDigestRow(t *testing.T) {
	s := newAuthDB(t)
	ctx := context.Background()
	seedUser(t, s, "alice", "pw", "admin")
	seedUser(t, s, "bob", "pw", "viewer")

	// Two sessions for alice (id=1), one for bob (id=2)
	aliceA, _, _ := s.Create(ctx, 1)
	aliceB, _, _ := s.Create(ctx, 1)
	bob, _, _ := s.Create(ctx, 2)

	// Delete alice's sessions
	err := s.DeleteForUser(ctx, 1)
	if err != nil {
		t.Fatalf("DeleteForUser: %v", err)
	}

	// alice's tokens should fail lookup
	_, _, err = s.lookup(ctx, aliceA)
	if err == nil {
		t.Error("lookup(aliceA after delete) should error")
	}
	_, _, err = s.lookup(ctx, aliceB)
	if err == nil {
		t.Error("lookup(aliceB after delete) should error")
	}

	// bob's token should still work
	_, _, err = s.lookup(ctx, bob)
	if err != nil {
		t.Errorf("lookup(bob after alice delete): %v", err)
	}

	// Check row count
	var count int
	s.db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE user_id=2`).Scan(&count)
	if count != 1 {
		t.Errorf("bob row count: %d, want 1", count)
	}
}

func TestSessions_CSRFStillEnforcedWithDigestStorage(t *testing.T) {
	s := newAuthDB(t)
	ctx := context.Background()
	seedUser(t, s, "alice", "pw", "admin")
	tok, csrf, err := s.Create(ctx, 1)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	handler := s.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	// POST with raw cookie and no CSRF header -> 403
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: tok})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("POST no CSRF: %d, want 403", rr.Code)
	}

	// POST with wrong CSRF -> 403
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: tok})
	req.Header.Set(csrfHeader, "wrong")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("POST wrong CSRF: %d, want 403", rr.Code)
	}

	// POST with correct CSRF -> 204
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: tok})
	req.Header.Set(csrfHeader, csrf)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("POST correct CSRF: %d, want 204", rr.Code)
	}

	// POST with stored digest as CSRF header -> 403
	var stored string
	s.db.DB.QueryRowContext(ctx, `SELECT token FROM sessions WHERE user_id=1`).Scan(&stored)
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: tok})
	req.Header.Set(csrfHeader, stored)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("POST with digest as CSRF: %d, want 403", rr.Code)
	}
}
