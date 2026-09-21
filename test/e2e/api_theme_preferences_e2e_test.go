//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// themePrefs mirrors the UserThemePreferences payload from
// specs/016-user-theme-customization/contracts/user-preferences-api.md §1.1.
// Optional fields are pointers so a JSON null (or an absent key) decodes as
// nil and can be told apart from an empty-but-present value.
type themePrefs struct {
	ThemeType        string       `json:"themeType"`
	PresetID         string       `json:"presetId"`
	AppearanceMode   string       `json:"appearanceMode"`
	CustomColors     *themeColors `json:"customColors"`
	CustomCSSEnabled bool         `json:"customCssEnabled"`
	CustomCSS        *string      `json:"customCss"`
	UpdatedAt        string       `json:"updatedAt"`
}

type themeColors struct {
	Accent  string `json:"accent"`
	Surface string `json:"surface"`
}

func decodeThemePrefs(t *testing.T, body []byte) themePrefs {
	t.Helper()
	var p themePrefs
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("decode preferences: %v\n%s", err, string(body))
	}
	return p
}

// getThemePrefs GETs /users/me/preferences and decodes the 200 payload.
func getThemePrefs(t *testing.T, cli *APIClient) themePrefs {
	t.Helper()
	resp, body, err := cli.Get("/users/me/preferences")
	if err != nil {
		t.Fatalf("GET /users/me/preferences: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /users/me/preferences: status=%d body=%s", resp.StatusCode, string(body))
	}
	return decodeThemePrefs(t, body)
}

// putThemePrefs issues a PUT /users/me/preferences and hands the raw
// status/body back so subtests can assert both successful round-trips
// and rejection messages.
func putThemePrefs(t *testing.T, cli *APIClient, body map[string]any) (*http.Response, []byte) {
	t.Helper()
	resp, rb, err := cli.Do(http.MethodPut, "/users/me/preferences", body)
	if err != nil {
		t.Fatalf("PUT /users/me/preferences: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp, rb
}

// getMeThemePrefs extracts the preferences object embedded in
// GET /users/me (contract §1.4).
func getMeThemePrefs(t *testing.T, cli *APIClient) themePrefs {
	t.Helper()
	resp, body, err := cli.Get("/users/me")
	if err != nil {
		t.Fatalf("GET /users/me: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /users/me: status=%d body=%s", resp.StatusCode, string(body))
	}
	var me struct {
		Preferences themePrefs `json:"preferences"`
	}
	if err := json.Unmarshal(body, &me); err != nil {
		t.Fatalf("decode /users/me: %v\n%s", err, string(body))
	}
	return me.Preferences
}

// requireRuleError asserts a 400 whose body carries a non-empty,
// rule-identifying message (contract §1.2). The message is read out of
// the {"error": "..."} envelope when the body is one, and matched
// against the raw body otherwise, so mustContain substrings apply either
// way.
func requireRuleError(t *testing.T, resp *http.Response, body []byte, mustContain ...string) {
	t.Helper()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%q", resp.StatusCode, string(body))
	}
	msg := string(body)
	if strings.TrimSpace(msg) == "" {
		t.Fatal("400 response has an empty body; want a rule-identifying message")
	}
	var env struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err == nil && env.Error != "" {
		msg = env.Error
	}
	for _, sub := range mustContain {
		if !strings.Contains(msg, sub) {
			t.Errorf("error message %q does not identify %q", msg, sub)
		}
	}
}

// TestAPI_ThemePreferences is the API-contract e2e for feature
// 016-user-theme-customization (contracts/user-preferences-api.md). A
// single viewer user exercises the real endpoints end to end: fresh-user
// Pink defaults, preset round-trips (including Legacy) through both
// GET /users/me/preferences and the preferences object embedded in
// GET /users/me, FR-012 retention across preset switches and overlay
// toggles, POST /users/me/preferences/reset semantics, FR-013
// sanitization rejections (plus the inline data:-URI acceptance),
// invalid enum/hex rejections, and 401s for unauthenticated callers.
//
// Migration provenance: migration 011 seeds preset_id='legacy' only for
// accounts that already existed when it ran. Every user on the e2e
// cluster — including e2e-admin — is created after that point (the API
// applies migrations at pod startup, before bootstrap-admin or any
// login can create a user), so no standard e2e bucket can manufacture a
// pre-migration user through the real API, and the INSERT..SELECT legacy
// seeding is verified by the api module's migration test instead
// (quickstart.md §3.1, TestUserThemePreferencesMigration). Here Legacy is
// covered as stored state: PUT to legacy, then read back through both
// GET surfaces.
//
// Budget: one e2e-admin login (to create the user) plus one login under
// a fresh UnixNano username — see the bucket comment in buckets.sh.
// t.Parallel(): the test mutates only its own viewer's preference rows
// and no shared config, so it is safe next to parallel neighbors.
func TestAPI_ThemePreferences(t *testing.T) {
	t.Parallel()

	envInstance.BootstrapAdmin(t, adminUsername, adminPassword)
	admin := envInstance.APIClient(t, adminUsername, adminPassword)
	defer admin.Close()

	// Viewer role on purpose: the contract grants preferences access to
	// ANY authenticated user ("Permissions: None"), so the test must not
	// need elevated roles.
	username, password, userID := envInstance.CreateUser(t, admin, "viewer", "e2e-theme-prefs")
	t.Cleanup(func() {
		r, _, _ := admin.Delete("/users/" + userID)
		if r != nil {
			r.Body.Close()
		}
	})

	user := envInstance.APIClient(t, username, password)
	defer user.Close()

	// fullPrefsBody builds the contract's §1.2 request shape with every
	// required field present and css as the customCss value. Each
	// sanitization case swaps in a poisoned css so the CSS rule under
	// test is the ONLY violation in an otherwise-valid payload.
	fullPrefsBody := func(css string) map[string]any {
		return map[string]any{
			"themeType":        "custom_colors",
			"presetId":         "pink",
			"appearanceMode":   "dark",
			"customColors":     map[string]string{"accent": "#10B981", "surface": "#121114"},
			"customCssEnabled": true,
			"customCss":        css,
		}
	}

	const dashboardCSS = ".dashboard-card { border-radius: 12px; }"

	t.Run("NewUserDefaultsToPink", func(t *testing.T) {
		// A user created after migration 011 has no user_preferences row;
		// both GET surfaces must synthesize the Pink default (§1.1 note).
		p := getThemePrefs(t, user)
		if p.PresetID != "pink" || p.ThemeType != "preset" || p.AppearanceMode != "system" {
			t.Fatalf("fresh user defaults = %+v, want preset/pink/system", p)
		}
		if p.CustomColors != nil || p.CustomCSS != nil || p.CustomCSSEnabled {
			t.Fatalf("fresh user must have no custom state: %+v", p)
		}
		if p.UpdatedAt == "" {
			t.Fatal("preferences payload missing updatedAt")
		}

		embedded := getMeThemePrefs(t, user)
		if embedded.PresetID != "pink" || embedded.ThemeType != "preset" ||
			embedded.CustomColors != nil || embedded.CustomCSS != nil || embedded.CustomCSSEnabled {
			t.Fatalf("/users/me embedded preferences = %+v, want Pink defaults", embedded)
		}
	})

	t.Run("PutRoundTripAndLegacyPreset", func(t *testing.T) {
		// Switch the base to Legacy (the preset migration 011 assigns to
		// pre-existing accounts) and read it back from the PUT response,
		// the dedicated endpoint, and the /users/me embedding (§1.4).
		resp, body := putThemePrefs(t, user, map[string]any{
			"themeType":        "preset",
			"presetId":         "legacy",
			"appearanceMode":   "dark",
			"customCssEnabled": false,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("PUT legacy preset: status=%d body=%s", resp.StatusCode, string(body))
		}
		if p := decodeThemePrefs(t, body); p.PresetID != "legacy" || p.AppearanceMode != "dark" || p.ThemeType != "preset" {
			t.Fatalf("PUT response = %+v, want preset/legacy/dark", p)
		}

		if p := getThemePrefs(t, user); p.PresetID != "legacy" || p.AppearanceMode != "dark" {
			t.Fatalf("stored preferences = %+v, want legacy/dark persisted (not just echoed)", p)
		}
		if embedded := getMeThemePrefs(t, user); embedded.PresetID != "legacy" || embedded.AppearanceMode != "dark" {
			t.Fatalf("/users/me embedded preferences = %+v, want legacy/dark", embedded)
		}
	})

	t.Run("CustomColorsAndCSSRoundTrip", func(t *testing.T) {
		resp, body := putThemePrefs(t, user, fullPrefsBody(dashboardCSS))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("PUT custom colors + css: status=%d body=%s", resp.StatusCode, string(body))
		}
		assertCustomState := func(p themePrefs, where string) {
			t.Helper()
			if p.ThemeType != "custom_colors" || p.PresetID != "pink" || p.AppearanceMode != "dark" || !p.CustomCSSEnabled {
				t.Fatalf("%s = %+v, want custom_colors/pink/dark/overlay-on", where, p)
			}
			if p.CustomColors == nil || p.CustomColors.Accent != "#10B981" || p.CustomColors.Surface != "#121114" {
				t.Fatalf("%s customColors = %+v, want #10B981/#121114", where, p.CustomColors)
			}
			if p.CustomCSS == nil || *p.CustomCSS != dashboardCSS {
				t.Fatalf("%s customCss = %v, want %q", where, p.CustomCSS, dashboardCSS)
			}
		}
		assertCustomState(decodeThemePrefs(t, body), "PUT response")
		assertCustomState(getThemePrefs(t, user), "GET /users/me/preferences")
		assertCustomState(getMeThemePrefs(t, user), "GET /users/me embedding")
	})

	t.Run("RetentionAcrossSwitches", func(t *testing.T) {
		// FR-012: preset switches and overlay toggles must never null
		// stored customColors/customCss — only reset does. Every PUT below
		// OMITS the optional custom fields, which per contract §1.2 keep
		// their stored values.
		assertRetained := func(p themePrefs, where string) {
			t.Helper()
			if p.CustomColors == nil || p.CustomColors.Accent != "#10B981" || p.CustomColors.Surface != "#121114" {
				t.Fatalf("%s: customColors lost across base/overlay change (FR-012): %+v", where, p.CustomColors)
			}
			if p.CustomCSS == nil || *p.CustomCSS != dashboardCSS {
				t.Fatalf("%s: customCss lost across base/overlay change (FR-012): %v", where, p.CustomCSS)
			}
		}

		// Baseline: customs stored with the overlay on (end state of the
		// previous subtest).
		// Preset switch only: pink -> legacy, base preset mode, overlay stays on.
		resp, body := putThemePrefs(t, user, map[string]any{
			"themeType":        "preset",
			"presetId":         "legacy",
			"appearanceMode":   "dark",
			"customCssEnabled": true,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("switch to legacy preset: status=%d body=%s", resp.StatusCode, string(body))
		}
		if p := decodeThemePrefs(t, body); p.PresetID != "legacy" || p.ThemeType != "preset" {
			t.Fatalf("switch response = %+v, want preset/legacy", p)
		}
		assertRetained(decodeThemePrefs(t, body), "preset switch response")

		// Overlay toggle off: customCss must survive the disable.
		resp, body = putThemePrefs(t, user, map[string]any{
			"themeType":        "preset",
			"presetId":         "legacy",
			"appearanceMode":   "dark",
			"customCssEnabled": false,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("disable overlay: status=%d body=%s", resp.StatusCode, string(body))
		}
		if p := decodeThemePrefs(t, body); p.CustomCSSEnabled {
			t.Fatalf("overlay still enabled after toggle off: %+v", p)
		}
		assertRetained(decodeThemePrefs(t, body), "overlay-off response")

		// Switch back: legacy -> pink, custom base mode, overlay on again.
		resp, body = putThemePrefs(t, user, map[string]any{
			"themeType":        "custom_colors",
			"presetId":         "pink",
			"appearanceMode":   "system",
			"customCssEnabled": true,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("switch back to pink: status=%d body=%s", resp.StatusCode, string(body))
		}
		assertRetained(decodeThemePrefs(t, body), "switch-back response")

		assertRetained(getThemePrefs(t, user), "GET after all switches")
	})

	t.Run("ResetToDefaults", func(t *testing.T) {
		// §1.3: reset is the ONLY operation that deletes stored customs.
		// The requested preset/appearanceMode are applied; everything
		// custom is nulled and the overlay is disabled.
		resp, body, err := user.Do(http.MethodPost, "/users/me/preferences/reset", map[string]any{
			"presetId":       "legacy",
			"appearanceMode": "dark",
		})
		if err != nil {
			t.Fatalf("POST reset: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("POST reset: status=%d body=%s", resp.StatusCode, string(body))
		}
		assertReset := func(p themePrefs, where string) {
			t.Helper()
			if p.CustomColors != nil || p.CustomCSS != nil || p.CustomCSSEnabled || p.ThemeType != "preset" {
				t.Fatalf("%s = %+v, want customs nulled, overlay off, base mode preset", where, p)
			}
		}
		p := decodeThemePrefs(t, body)
		assertReset(p, "reset response")
		if p.PresetID != "legacy" || p.AppearanceMode != "dark" {
			t.Fatalf("reset response = %+v, want requested preset legacy/dark restored", p)
		}

		stored := getThemePrefs(t, user)
		assertReset(stored, "GET after reset")
		if stored.PresetID != "legacy" || stored.AppearanceMode != "dark" {
			t.Fatalf("stored preferences after reset = %+v, want legacy/dark", stored)
		}

		// Empty body: preset_id/appearance_mode default to the CURRENT
		// values (contract §1.3), so a second reset must keep legacy/dark.
		resp, body, err = user.Do(http.MethodPost, "/users/me/preferences/reset", nil)
		if err != nil {
			t.Fatalf("POST reset (no body): %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("POST reset (no body): status=%d body=%s", resp.StatusCode, string(body))
		}
		noBody := decodeThemePrefs(t, body)
		assertReset(noBody, "no-body reset response")
		if noBody.PresetID != "legacy" || noBody.AppearanceMode != "dark" {
			t.Fatalf("no-body reset = %+v, want current preset legacy/dark kept", noBody)
		}
	})

	t.Run("SanitizationRejects", func(t *testing.T) {
		// FR-013 / data-model §5.6: each poisonous customCss must 400 with
		// a message naming the offending rule, and must not alter stored
		// state. Baseline first so the post-table GET proves the 400s were
		// side-effect free.
		resp, body := putThemePrefs(t, user, fullPrefsBody(dashboardCSS))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("baseline PUT: status=%d body=%s", resp.StatusCode, string(body))
		}

		cases := []struct {
			name string
			css  string
			want []string
		}{
			{
				name: "ImportRule",
				css:  `@import url("https://evil.example.com/track.css");`,
				want: []string{"@import"},
			},
			{
				name: "ExternalHTTPSUrl",
				css:  `.banner { background: url("https://fonts.example.com/x.css"); }`,
				want: []string{"fonts.example.com"},
			},
			{
				name: "ProtocolRelativeUrl",
				css:  `.pixel { background: url("//evil.example.com/pixel.png"); }`,
				want: []string{"evil.example.com"},
			},
			{
				// Valid, balanced CSS whose only violation is size.
				name: "Over32KB",
				css:  "/* " + strings.Repeat("x", 33000) + " */",
			},
			{
				name: "ScriptTag",
				css:  `.ok { color: red; } <script>alert("x")</script>`,
				want: []string{"script"},
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				resp, body := putThemePrefs(t, user, fullPrefsBody(tc.css))
				requireRuleError(t, resp, body, tc.want...)
			})
		}

		// None of the rejected PUTs may have touched the stored row.
		if p := getThemePrefs(t, user); p.CustomCSS == nil || *p.CustomCSS != dashboardCSS {
			t.Fatalf("rejected CSS mutated stored preferences: %+v", p)
		}
	})

	t.Run("DataURIAccepted", func(t *testing.T) {
		// Inline data: URIs are the sanctioned exception to the external
		// url() ban and must round-trip (§1.2 validation rules).
		const dataCSS = `.logo { background-image: url("data:image/png;base64,iVBORw0KGgoAAAANSUhEUg="); }`
		resp, body := putThemePrefs(t, user, fullPrefsBody(dataCSS))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("PUT data: URI css: status=%d body=%s", resp.StatusCode, string(body))
		}
		if p := getThemePrefs(t, user); p.CustomCSS == nil || *p.CustomCSS != dataCSS {
			t.Fatalf("data: URI css not stored verbatim: %+v", p)
		}
	})

	t.Run("InvalidValuesRejected", func(t *testing.T) {
		// Enums are strict (§1.2), hex must be ^#([0-9a-fA-F]{6})$,
		// required fields must be present. All 400s, none may mutate state.
		mutated := func(mut func(map[string]any)) map[string]any {
			m := fullPrefsBody(dashboardCSS)
			delete(m, "customCss")
			delete(m, "customColors")
			mut(m)
			return m
		}
		cases := []struct {
			name string
			body map[string]any
		}{
			{"BadHexDigit", mutated(func(m map[string]any) {
				m["customColors"] = map[string]string{"accent": "#XYZ012", "surface": "#121114"}
			})},
			{"ShortHex", mutated(func(m map[string]any) {
				m["customColors"] = map[string]string{"accent": "#12345", "surface": "#121114"}
			})},
			{"BadThemeType", mutated(func(m map[string]any) { m["themeType"] = "neon" })},
			{"BadPresetID", mutated(func(m map[string]any) { m["presetId"] = "plaid" })},
			{"BadAppearanceMode", mutated(func(m map[string]any) { m["appearanceMode"] = "auto" })},
			// Required fields (appearanceMode, customCssEnabled) missing.
			{"MissingRequired", map[string]any{"themeType": "preset", "presetId": "pink"}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				resp, body := putThemePrefs(t, user, tc.body)
				requireRuleError(t, resp, body)
			})
		}

		// Stored state (the data: URI css from the previous subtest) must
		// be untouched by every rejection above.
		const dataCSS = `.logo { background-image: url("data:image/png;base64,iVBORw0KGgoAAAANSUhEUg="); }`
		if p := getThemePrefs(t, user); p.CustomCSS == nil || *p.CustomCSS != dataCSS {
			t.Fatalf("invalid PUTs mutated stored preferences: %+v", p)
		}
	})

	t.Run("UnauthenticatedIsRejected", func(t *testing.T) {
		// No session cookie at all: every preferences surface must 401
		// (contract §1.1–§1.3). Borrow the user's port-forward BaseURL and
		// issue raw requests with a cookie-less client, like
		// TestAPI_LoginPrivacy does.
		raw := &http.Client{Timeout: user.HTTP.Timeout}
		cases := []struct {
			name   string
			method string
			path   string
			body   string
		}{
			{"Get", http.MethodGet, "/users/me/preferences", ""},
			{"Put", http.MethodPut, "/users/me/preferences",
				`{"themeType":"preset","presetId":"pink","appearanceMode":"system","customCssEnabled":false}`},
			{"Reset", http.MethodPost, "/users/me/preferences/reset", ""},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				req, err := http.NewRequestWithContext(t.Context(), tc.method, user.BaseURL+tc.path, strings.NewReader(tc.body))
				if err != nil {
					t.Fatalf("build %s request: %v", tc.name, err)
				}
				if tc.body != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				resp, err := raw.Do(req)
				if err != nil {
					t.Fatalf("%s %s: %v", tc.method, tc.path, err)
				}
				defer func() { _ = resp.Body.Close() }()
				if resp.StatusCode != http.StatusUnauthorized {
					t.Fatalf("%s %s: status=%d, want 401", tc.method, tc.path, resp.StatusCode)
				}
			})
		}
	})
}
