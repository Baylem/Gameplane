package controlplane

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestKeyPublicationWaitsForDirectorySync(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "panel.key")
	wantErr := errors.New("directory sync failed")
	called := false
	key, err := loadKeyWithSync(path, false, func(got string) error {
		called = true
		if got != dir {
			t.Fatalf("synced %q instead of key directory", got)
		}
		info, statErr := os.Stat(path)
		if statErr != nil || info.Size() != 32 {
			t.Fatal("key was not fully published before directory sync")
		}
		return wantErr
	})
	if !called || !errors.Is(err, wantErr) || key != nil {
		t.Fatalf("unsynced key became usable: called=%v err=%v", called, err)
	}
	// A sync failure must never remove or replace the potentially published key.
	if info, err := os.Stat(path); err != nil || info.Size() != 32 {
		t.Fatal("lost published key after directory sync failure")
	}
}

func TestNestedKeyDirectoriesAreDurablyPublished(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "private", "keys", "panel.key")
	var synced []string
	key, err := loadKeyWithSync(path, false, func(dir string) error {
		synced = append(synced, dir)
		return syncKeyDirectory(dir)
	})
	if err != nil || len(key) != 32 {
		t.Fatalf("create key: %v", err)
	}
	want := []string{base, filepath.Join(base, "private"), filepath.Dir(path)}
	if len(synced) != len(want) {
		t.Fatalf("directory sync count = %d, want %d", len(synced), len(want))
	}
	for i := range want {
		if synced[i] != want[i] {
			t.Fatalf("directory sync %d = %q, want %q", i, synced[i], want[i])
		}
	}
}

func TestKeyDirectorySyncFailurePreventsPublication(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "private", "panel.key")
	wantErr := errors.New("parent directory sync failed")
	if key, err := loadKeyWithSync(path, false, func(string) error { return wantErr }); key != nil || !errors.Is(err, wantErr) {
		t.Fatalf("accepted undurable directory: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("published key after directory sync failed")
	}
}
