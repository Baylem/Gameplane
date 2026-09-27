package controller

import (
	"testing"
)

func TestRestoredRefName_UniqueAndUnderLimit(t *testing.T) {
	tests := []struct {
		serverName string
		refName    string
		wantUnique bool
	}{
		{"server1", "secret1", true},
		{"server1", "secret2", true},
		{"server1", "secret1", true}, // Same ref name should produce consistent name
		{"server-with-very-long-name", "secret-with-very-long-name", true},
	}

	names := make(map[string]bool)

	for _, tt := range tests {
		name := restoredRefName(tt.serverName, tt.refName)

		// Check length
		if len(name) > maxObjectNameLength {
			t.Errorf("restoredRefName(%q, %q) = %q (len %d), exceeded limit %d",
				tt.serverName, tt.refName, name, len(name), maxObjectNameLength)
		}

		// Check format
		if len(name) == 0 {
			t.Errorf("restoredRefName(%q, %q) returned empty name", tt.serverName, tt.refName)
		}

		if tt.wantUnique {
			if names[name] {
				t.Errorf("restoredRefName(%q, %q) = %q, already used", tt.serverName, tt.refName, name)
			}
			names[name] = true
		}
	}

	// Test consistency: same inputs produce same output
	name1 := restoredRefName("server", "secret")
	name2 := restoredRefName("server", "secret")
	if name1 != name2 {
		t.Errorf("restoredRefName not consistent: %q vs %q", name1, name2)
	}
}

func TestHashString_Consistent(t *testing.T) {
	h1 := hashString("test-string")
	h2 := hashString("test-string")
	if h1 != h2 {
		t.Errorf("hashString not consistent: %d vs %d", h1, h2)
	}

	// Different strings should (likely) produce different hashes
	h3 := hashString("different-string")
	if h1 == h3 {
		t.Logf("hashString collision (unlikely but possible): %d", h1)
	}
}

// TestRestoredRefName_NoPanicOnSmallHash is a regression test: hex-encoding
// the hash with a plain "%x" drops leading zero nibbles, so any hash value
// under 0x1_0000_0000_0000 yields a string shorter than 12 characters and
// panics on the `[:12]` slice below it. hashString("") deterministically
// hits this — it's the raw djb2 seed 5381, "1505" in hex, only 4 digits —
// so an empty origRefName reliably reproduced the panic pre-fix.
// "%012x" zero-pads first, making the slice always safe.
func TestRestoredRefName_NoPanicOnSmallHash(t *testing.T) {
	for _, refName := range []string{"", "a", "0"} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("restoredRefName(%q, %q) panicked: %v", "server", refName, r)
				}
			}()
			name := restoredRefName("server", refName)
			if len(name) == 0 {
				t.Errorf("restoredRefName(%q, %q) returned empty name", "server", refName)
			}
		}()
	}
}

// planOwnedRefCopies, ensureOwnedRefCopies and ensureRestoredRefs are
// covered end to end by the envtest tests; the requeue behaviour on
// transient errors is covered with a fake client in
// restore_volumesnapshot_requeue_test.go.
