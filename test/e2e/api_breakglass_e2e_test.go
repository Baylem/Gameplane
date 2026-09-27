//go:build e2e

package e2e

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

// TestAPI_BootstrapAdminForceEndsExistingSessions: a forced reset through
// the bootstrap-admin break-glass command ends the reset account's
// existing sessions, the same way the dashboard password reset does.
//
// It uses its own throwaway account, never e2e-admin (resetting e2e-admin
// would end the sessions other tests in the job hold). Budget: one
// e2e-admin login (for cleanup) plus one local login as the throwaway
// account (a fresh per-username bucket, one slot of the job's shared
// per-IP budget), plus two kubectl exec calls.
func TestAPI_BootstrapAdminForceEndsExistingSessions(t *testing.T) {
	t.Parallel()

	// Generate random credentials with fixed prefixes.
	username := randomUsername("e2e-breakglass-reset")
	firstPassword := randomPassword("e2e-breakglass-first-password")
	resetPassword := randomPassword("e2e-breakglass-reset-password")

	// Let the shared e2e-admin bootstrap (once per process) finish first,
	// so this test's bootstrap-admin execs never overlap with it.
	envInstance.BootstrapAdmin(t, adminUsername, adminPassword)

	// Admin client used by the deferred cleanup below.
	admin := envInstance.APIClient(t, adminUsername, adminPassword)
	defer admin.Close()

	// Delete the throwaway account while the test context (and so the
	// port-forward) is still live. A t.Cleanup callback would run after
	// t.Context() is cancelled and fail with "connection refused".
	defer func() {
		// List users to find the throwaway account's ID.
		resp, body, err := admin.Get("/users")
		if err != nil {
			t.Errorf("list users for cleanup: %v", err)
			return
		}
		resp.Body.Close()

		var users []struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		}
		if err := json.Unmarshal(body, &users); err != nil {
			t.Errorf("decode users list for cleanup: %v", err)
			return
		}

		// Find the throwaway user by username.
		var userID int64
		for _, u := range users {
			if u.Username == username {
				userID = u.ID
				break
			}
		}
		if userID == 0 {
			t.Errorf("cleanup: throwaway user %q not found in user list", username)
			return
		}

		// Delete the user.
		delResp, _, err := admin.Delete("/users/" + strconv.FormatInt(userID, 10))
		if delResp != nil {
			delResp.Body.Close()
		}
		if err != nil {
			t.Errorf("delete user %q: %v", username, err)
			return
		}
	}()

	bootstrap := func(password string) {
		t.Helper()
		out, err := envInstance.KubectlWithStdin(t.Context(), password+"\n",
			"exec", "-i", "-n", "gameplane-system", "deploy/gameplane-api", "--",
			"/api", "bootstrap-admin",
			"--username="+username,
			"--password-stdin",
			"--force",
		)
		if err != nil {
			t.Fatalf("bootstrap-admin: %v\n%s", err, out)
		}
	}

	// --force on the first call too, so a rerun against a reused cluster
	// resets the account instead of failing on "already exists".
	bootstrap(firstPassword)

	cli := envInstance.APIClient(t, username, firstPassword)
	defer cli.Close()

	resp, body, err := cli.Get("/users/me")
	if err != nil {
		t.Fatalf("baseline /users/me: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("baseline /users/me: status=%d body=%s", resp.StatusCode, string(body))
	}

	bootstrap(resetPassword)

	resp, body, err = cli.Get("/users/me")
	if err != nil {
		t.Fatalf("post-reset /users/me: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-reset /users/me: status=%d body=%s, want 401 (session ended by the reset)",
			resp.StatusCode, string(body))
	}
}

// randomUsername generates a username with the given prefix followed by
// 16 bytes of random hex (32 hex characters), respecting the identifier
// regex constraint of max 64 characters total.
func randomUsername(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + "-" + hex.EncodeToString(b)
}

// randomPassword generates a password with the given prefix followed by
// 16 bytes of random hex (32 hex characters).
func randomPassword(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + "-" + hex.EncodeToString(b)
}
