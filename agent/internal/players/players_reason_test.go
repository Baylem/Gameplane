package players

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSanitizeReason_SharedPolicy pins that the moderation reason applies the
// gameaction input-character policy: plain text passes; command separators,
// quotes, backslashes and control characters are rejected.
func TestSanitizeReason_SharedPolicy(t *testing.T) {
	const (
		wantControl = "reason must not contain control characters"
		wantMeta    = "reason must not contain command separators, quotes or backslashes"
	)
	allowed := []string{
		"",
		"griefing",
		"x-ray (fly hack) #2: 100% sure, ok?",
		"café 你好",
		"spaces are fine",
	}
	for _, in := range allowed {
		got, err := sanitizeReason(in)
		if err != nil {
			t.Errorf("sanitizeReason(%q) unexpected error: %v", in, err)
		}
		if got != in {
			t.Errorf("sanitizeReason(%q) = %q, want unchanged", in, got)
		}
	}

	rejected := []struct {
		name, in, want string
	}{
		{"semicolon", "bad; stop", wantMeta},
		{"ampersand", "bad & stop", wantMeta},
		{"pipe", "bad | stop", wantMeta},
		{"dollar", "bad $x", wantMeta},
		{"backtick", "bad `x`", wantMeta},
		{"backslash", `bad\n`, wantMeta},
		{"double quote", `say "hi"`, wantMeta},
		{"single quote", "didn't listen", wantMeta},
		{"newline", "a\nstop", wantControl},
		{"carriage return", "a\rstop", wantControl},
		{"nul", "a\x00b", wantControl},
		{"escape", "a\x1bb", wantControl},
		{"tab", "a\tb", wantControl},
		{"del", "a\x7fb", wantControl},
		{"control beats separator", "a;b\nc", wantControl},
		// Validation runs on the full input before the 256-byte truncation,
		// so a bad character past the cap is still rejected.
		{"separator past the cap", strings.Repeat("x", 300) + ";", wantMeta},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sanitizeReason(tc.in)
			if err == nil {
				t.Fatalf("sanitizeReason(%q) = %q, want error", tc.in, got)
			}
			if err.Error() != tc.want {
				t.Errorf("sanitizeReason(%q) error = %q, want %q", tc.in, err.Error(), tc.want)
			}
		})
	}
}

// TestModerationReasonPolicy_HTTP drives kick and ban over HTTP: a rejected
// reason is a 400 with the documented message and never reaches RCON; an
// allowed reason is rendered into the command.
func TestModerationReasonPolicy_HTTP(t *testing.T) {
	rc := &fakeRcon{}
	srv := httptest.NewServer(newTestRouter(t, "minecraft-java", rc))
	defer srv.Close()

	bad := []struct{ name, reason, wantMsg string }{
		{"separator", "x; op alice", "command separators"},
		{"quote", `x "y"`, "command separators"},
		{"apostrophe", "x 'y'", "command separators"},
		{"backslash", `x\y`, "command separators"},
		{"newline", "x\nop alice", "control characters"},
		{"nul", "x\x00y", "control characters"},
	}
	for _, path := range []string{"/players/kick", "/players/ban"} {
		for _, tc := range bad {
			t.Run(path+"/"+tc.name, func(t *testing.T) {
				rc.last = ""
				status, body := doJSON(t, srv, "POST", path, modReq{Name: "alice", Reason: tc.reason})
				if status != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body %s)", status, body)
				}
				if !strings.Contains(string(body), tc.wantMsg) {
					t.Errorf("body %s does not mention %q", body, tc.wantMsg)
				}
				if rc.last != "" {
					t.Errorf("rcon must not be invoked, got %q", rc.last)
				}
			})
		}
	}

	rc.last = ""
	status, body := doJSON(t, srv, "POST", "/players/kick", modReq{Name: "alice", Reason: "x-ray (fly), 100%"})
	if status != http.StatusOK {
		t.Fatalf("allowed reason: status = %d, want 200 (body %s)", status, body)
	}
	if rc.last != "kick alice x-ray (fly), 100%" {
		t.Errorf("rcon called with %q", rc.last)
	}
}
