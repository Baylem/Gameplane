//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// i18nLanguagePrefs is the language slice of GET /users/me/preferences.
type i18nLanguagePrefs struct {
	Language *string `json:"language"`
}

// i18nGetLanguage reads the caller's saved language from
// GET /users/me/preferences. A nil result means "no saved choice".
func i18nGetLanguage(t *testing.T, cli *APIClient) *string {
	t.Helper()
	resp, body, err := cli.Get("/users/me/preferences")
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		t.Fatalf("GET /users/me/preferences: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /users/me/preferences: status=%d body=%s", resp.StatusCode, string(body))
	}
	var prefs i18nLanguagePrefs
	if err := json.Unmarshal(body, &prefs); err != nil {
		t.Fatalf("decode preferences: %v\n%s", err, string(body))
	}
	return prefs.Language
}

// TestAPI_I18n_LanguagePref proves PUT /users/me/language (spec 019, T035):
// a listed tag is saved and reads back through the preferences payload, null
// clears it, and a rejected tag returns the registry problem document without
// writing anything. It acts only on the admin's own account and resets the
// language to null on cleanup, so later live specs run in English again.
//
// Budget: one e2e-admin login. It lives in the operator bucket for the login
// budget, not by subject, the same precedent as the EventStream test there.
func TestAPI_I18n_LanguagePref(t *testing.T) {
	t.Parallel()

	envInstance.BootstrapAdmin(t, adminUsername, adminPassword)
	admin := envInstance.APIClient(t, adminUsername, adminPassword)
	t.Cleanup(admin.Close)
	// A defer, not t.Cleanup: the client's port-forward is bound to
	// t.Context(), which is canceled before cleanups run, so the reset would
	// never reach the API from a cleanup.
	defer func() {
		r, _, _ := admin.Do(http.MethodPut, "/users/me/language", map[string]any{"language": nil})
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	// A listed tag is saved and echoed back by the preferences payload.
	resp, body, err := admin.Do(http.MethodPut, "/users/me/language", map[string]any{"language": "fr"})
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		t.Fatalf("PUT /users/me/language fr: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT /users/me/language fr: status=%d body=%s", resp.StatusCode, string(body))
	}
	if got := i18nGetLanguage(t, admin); got == nil || *got != "fr" {
		t.Fatalf("language after PUT fr = %v, want fr", got)
	}

	// Null clears the saved choice.
	resp, body, err = admin.Do(http.MethodPut, "/users/me/language", map[string]any{"language": nil})
	if resp != nil {
		r := resp
		defer func() { _ = r.Body.Close() }()
	}
	if err != nil {
		t.Fatalf("PUT /users/me/language null: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT /users/me/language null: status=%d body=%s", resp.StatusCode, string(body))
	}
	if got := i18nGetLanguage(t, admin); got != nil {
		t.Fatalf("language after PUT null = %q, want nil", *got)
	}

	// Plain-text callers keep the byte-identical body for a bad body shape.
	status, _, raw := rawDo(t, admin, http.MethodPut, "/users/me/language", map[string]any{}, "")
	if status != http.StatusBadRequest || strings.TrimSpace(string(raw)) != "language is required" {
		t.Errorf("PUT /users/me/language without key: status=%d body=%q, want 400 \"language is required\"", status, string(raw))
	}
	status, _, raw = rawDo(t, admin, http.MethodPut, "/users/me/language", map[string]any{"language": 5}, "")
	if status != http.StatusBadRequest || strings.TrimSpace(string(raw)) != "language must be a string or null" {
		t.Errorf("PUT /users/me/language non-string: status=%d body=%q, want 400 \"language must be a string or null\"", status, string(raw))
	}

	// An unlisted tag is rejected with the localizable problem document.
	const unknown = "xx"
	status, ct, raw := rawDo(t, admin, http.MethodPut, "/users/me/language", map[string]any{"language": unknown}, i18nProblemAccept)
	if status != http.StatusBadRequest {
		t.Fatalf("PUT /users/me/language xx (problem): status=%d body=%q", status, string(raw))
	}
	doc := decodeProblemBody(t, "PUT /users/me/language xx", ct, raw)
	requireProblemFields(t, "PUT /users/me/language xx", doc, http.StatusBadRequest,
		"server.i18n.unknown_language", "unknown language: xx")
	args, ok := doc["args"].(map[string]any)
	if !ok || args["language"] != unknown {
		t.Errorf("PUT /users/me/language xx: args=%v, want language=%q", doc["args"], unknown)
	}

	// Plain-text callers get the same message as plain text.
	status, _, raw = rawDo(t, admin, http.MethodPut, "/users/me/language", map[string]any{"language": unknown}, "")
	if status != http.StatusBadRequest || strings.TrimSpace(string(raw)) != "unknown language: xx" {
		t.Errorf("PUT /users/me/language xx (plain): status=%d body=%q, want 400 \"unknown language: xx\"", status, string(raw))
	}

	// The rejected writes must not have saved anything.
	if got := i18nGetLanguage(t, admin); got != nil {
		t.Fatalf("language after rejected PUT = %q, want nil", *got)
	}
}
