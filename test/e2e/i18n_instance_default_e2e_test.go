//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestAPI_I18n_InstanceDefault proves the instance default language (spec 019,
// T092): an admin can set it through PUT /admin/config/i18n, the unauthenticated
// GET /auth/locale reflects it, an unlisted or empty value is rejected without
// writing, and a viewer cannot change it. The default is process-wide, so the
// test holds i18nInstanceDefaultMu for its whole body and restores the prior
// value on cleanup.
//
// Budget: one e2e-admin login plus one viewer login, in the multicluster
// bucket for the login budget (see buckets.sh).
func TestAPI_I18n_InstanceDefault(t *testing.T) {
	t.Parallel()

	envInstance.BootstrapAdmin(t, adminUsername, adminPassword)
	admin := envInstance.APIClient(t, adminUsername, adminPassword)
	// Close is registered first so it runs last: every cleanup below uses admin.
	t.Cleanup(admin.Close)

	i18nInstanceDefaultMu.Lock()
	t.Cleanup(i18nInstanceDefaultMu.Unlock)

	username, password, id := envInstance.CreateUser(t, admin, "viewer", "e2e-i18n-default")
	t.Cleanup(func() {
		r, _, _ := admin.Delete("/users/" + id)
		if r != nil {
			_ = r.Body.Close()
		}
	})
	viewer := envInstance.APIClient(t, username, password)
	t.Cleanup(viewer.Close)

	// Restore the prior i18n section, or en when it was absent.
	prior, hadPrior := i18nConfigSection(t, admin, "i18n")
	t.Cleanup(func() {
		body := prior
		if !hadPrior {
			body = json.RawMessage(`{"defaultLanguage":"en"}`)
		}
		r, _, _ := admin.Do(http.MethodPut, "/admin/config/i18n", body)
		if r != nil {
			_ = r.Body.Close()
		}
	})

	port, stop := envInstance.PortForward(t, "gameplane-system", "svc/gameplane-api", 80)
	defer stop()

	// The admin sets a listed default; the unauthenticated endpoint reflects it.
	resp, body, err := admin.Do(http.MethodPut, "/admin/config/i18n", map[string]string{"defaultLanguage": "fr"})
	if err != nil {
		t.Fatalf("PUT /admin/config/i18n fr: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /admin/config/i18n fr: status=%d body=%s", resp.StatusCode, string(body))
	}
	if got := i18nAuthLocale(t, port); got != "fr" {
		t.Fatalf("/auth/locale defaultLanguage=%q after PUT fr, want fr", got)
	}

	// An unlisted tag is rejected with the registry problem document.
	status, ct, raw := rawDo(t, admin, http.MethodPut, "/admin/config/i18n",
		map[string]string{"defaultLanguage": "tlh"}, i18nProblemAccept)
	if status != http.StatusBadRequest {
		t.Fatalf("PUT /admin/config/i18n tlh: status=%d body=%q", status, string(raw))
	}
	doc := decodeProblemBody(t, "PUT /admin/config/i18n tlh", ct, raw)
	requireProblemFields(t, "PUT /admin/config/i18n tlh", doc, http.StatusBadRequest,
		"server.i18n.unlisted_language", "language is not available: tlh")
	if args, ok := doc["args"].(map[string]any); !ok || args["language"] != "tlh" {
		t.Errorf("PUT /admin/config/i18n tlh: args=%v, want language=tlh", doc["args"])
	}

	// An empty default is rejected as plain text.
	status, _, raw = rawDo(t, admin, http.MethodPut, "/admin/config/i18n",
		map[string]string{"defaultLanguage": ""}, "")
	if status != http.StatusBadRequest || strings.TrimSpace(string(raw)) != "defaultLanguage is required" {
		t.Errorf("PUT /admin/config/i18n empty: status=%d body=%q, want 400 \"defaultLanguage is required\"", status, string(raw))
	}

	// Neither rejected write changed the default.
	if got := i18nAuthLocale(t, port); got != "fr" {
		t.Fatalf("/auth/locale defaultLanguage=%q after rejected writes, want fr", got)
	}

	// A viewer cannot change the instance default.
	status, _, raw = rawDo(t, viewer, http.MethodPut, "/admin/config/i18n",
		map[string]string{"defaultLanguage": "en"}, "")
	if status != http.StatusForbidden {
		t.Fatalf("viewer PUT /admin/config/i18n: status=%d body=%q, want 403", status, string(raw))
	}
	if got := i18nAuthLocale(t, port); got != "fr" {
		t.Fatalf("/auth/locale defaultLanguage=%q after viewer 403, want fr", got)
	}
}
