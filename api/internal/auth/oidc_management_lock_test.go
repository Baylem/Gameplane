package auth

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestOIDCResyncWaitsForConcurrentUserManagement(t *testing.T) {
	store := newAuthDB(t)
	ctx := context.Background()
	for _, name := range []string{"oidc-admin", "other-admin"} {
		if _, err := store.DB.ExecContext(ctx,
			`INSERT INTO users(username, display_name, email, role) VALUES (?, ?, ?, 'admin')`, name, name, name+"@example.com"); err != nil {
			t.Fatal(err)
		}
	}
	var userID int64
	if err := store.DB.QueryRowContext(ctx, `SELECT id FROM users WHERE username = 'oidc-admin'`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	o := &OIDC{}
	o.AttachStore(store)

	// A manual management operation owns the guard while it removes the
	// other administrator. OIDC must wait before reading the manager count.
	unlock := store.LockUserManagement()
	var once sync.Once
	release := func() { once.Do(unlock) }
	defer release()
	type result struct {
		outcome *RoleAssignmentOutcome
		err     error
	}
	started := make(chan struct{})
	done := make(chan result, 1)
	go func() {
		close(started)
		outcome, err := o.syncUserRole(ctx, userID, "viewer", "viewers")
		done <- result{outcome, err}
	}()
	<-started
	select {
	case got := <-done:
		t.Fatalf("resync completed while user management was locked: outcome=%+v err=%v", got.outcome, got.err)
	case <-time.After(100 * time.Millisecond):
		// Bound this negative assertion; do not release the guard early.
	}
	if _, err := store.DB.ExecContext(ctx, `DELETE FROM users WHERE username = 'other-admin'`); err != nil {
		t.Fatal(err)
	}
	release()
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatal(got.err)
		}
		if got.outcome.Applied {
			t.Fatal("OIDC demoted the last remaining manager")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("resync did not finish after releasing the management guard")
	}
	count, err := store.UserManagerCount(ctx)
	if err != nil || count != 1 {
		t.Fatalf("manager count=%d err=%v, want one manager", count, err)
	}
}
