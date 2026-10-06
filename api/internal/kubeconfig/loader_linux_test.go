//go:build linux

package kubeconfig

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// A token-file read would block on the empty FIFO. Rejection must happen before
// client-go can open it, even when the selected user has no embedded token.
func TestRESTConfigDoesNotReadTokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token-pipe")
	if err := unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Release any blocked reader if a future regression causes this test to fail.
		fd, err := unix.Open(path, unix.O_RDWR|unix.O_NONBLOCK, 0600)
		if err == nil {
			_, _ = unix.Write(fd, []byte("process-secret"))
			_ = unix.Close(fd)
		}
	})
	cfg := testConfig()
	cfg.AuthInfos["selected"].Token = ""
	cfg.AuthInfos["selected"].TokenFile = path
	data := configBytes(t, cfg)
	done := make(chan error, 1)
	go func() {
		result, err := RESTConfig(data)
		if result != nil {
			done <- fmt.Errorf("unexpected REST config from a token file")
			return
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "tokenFile") {
			t.Fatalf("expected tokenFile policy rejection, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("kubeconfig parsing tried to read the token file")
	}
}
