package files

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

const identityTestEnv = "GAMEPLANE_FILE_IDENTITY_TEST"

// TestFileAccessAcrossUIDs needs root only to prepare real ownership and launch
// child processes with distinct credentials. All file operations under test run
// without capabilities as UID 65532 (agent), UID 1000 (game), or ACL readers.
// CI runs this single compiled test explicitly under sudo; ordinary non-root
// test suites skip the harness and never invoke sudo themselves.
func TestFileAccessAcrossUIDs(t *testing.T) {
	if role := os.Getenv(identityTestEnv); role != "" {
		runFileIdentityChild(t, role)
		return
	}
	if os.Geteuid() != 0 {
		t.Skip("root harness required to launch distinct unprivileged UIDs")
	}
	// Go's normal test binary and TempDir may sit under private directories.
	// Copy the executable to a traversable, root-owned directory for children.
	base, err := os.MkdirTemp("/tmp", "gameplane-file-identities-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	if err := os.Chmod(base, 0o755); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	childBinary := filepath.Join(base, "files.test")
	if err := os.WriteFile(childBinary, binary, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(childBinary, 0o755); err != nil {
		t.Fatal(err)
	}
	const sharedGroup = 5000
	for _, method := range []string{"write", "upload"} {
		for _, scenario := range []string{"0600", "0644", "extended-acl", "group-failure", "acl-failure"} {
			t.Run(method+"/"+scenario, func(t *testing.T) {
				root := filepath.Join(base, method+"-"+scenario)
				if err := os.Mkdir(root, 0o775); err != nil {
					t.Fatal(err)
				}
				if err := os.Chown(root, 0, sharedGroup); err != nil {
					t.Fatal(err)
				}
				// Defeat the harness's umask; only the shared group can publish.
				if err := os.Chmod(root, 0o775); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(root, "target.txt")
				mode := os.FileMode(0o600)
				if scenario == "0644" {
					mode = 0o644
				}
				if err := os.WriteFile(target, []byte("original"), mode); err != nil {
					t.Fatal(err)
				}
				gid := sharedGroup
				if scenario == "group-failure" {
					gid++ // Deliberately absent from the agent's groups.
				}
				if err := os.Chown(target, 1000, gid); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(target, mode); err != nil {
					t.Fatal(err)
				}
				if scenario == "extended-acl" {
					raw := encodeACL([]aclEntry{{aclUserObj, 6, aclUndefined}, {aclUser, 7, 2000}, {aclGroupObj, 7, aclUndefined}, {aclGroup, 6, 3000}, {aclMask, 4, aclUndefined}, {aclOther, 0, aclUndefined}})
					// A missing ACL-capable filesystem is a CI failure, not a skip.
					if err := unix.Setxattr(target, accessACLName, raw, 0); err != nil {
						t.Fatal(err)
					}
				}
				var before unix.Stat_t
				if err := unix.Stat(target, &before); err != nil {
					t.Fatal(err)
				}
				run := func(role string, uid, gid uint32, groups []uint32, expected string) {
					t.Helper()
					cmd := exec.CommandContext(t.Context(), childBinary, "-test.run=^TestFileAccessAcrossUIDs$", "-test.v")
					cmd.Env = append(os.Environ(), identityTestEnv+"="+role, "GAMEPLANE_FILE_ROOT="+root, "GAMEPLANE_FILE_METHOD="+method, "GAMEPLANE_FILE_SCENARIO="+scenario, "GAMEPLANE_FILE_EXPECTED="+expected)
					cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: gid, Groups: groups}}
					if out, err := cmd.CombinedOutput(); err != nil {
						t.Fatalf("%s UID %d GID %d groups %v: %v\n%s", role, uid, gid, groups, err, out)
					}
				}
				// Prove effective ACL permissions before and after replacement.
				checkReaders := func(expected string) {
					if scenario == "extended-acl" {
						run("read-only", 2000, 2000, nil, expected)            // named user
						run("read-only", 4000, 4000, []uint32{3000}, expected) // named group
					}
					groupRole, otherRole := "denied", "denied"
					if scenario == "0644" || scenario == "extended-acl" {
						groupRole = "read-only"
					}
					if scenario == "0644" {
						otherRole = "read-only"
					}
					run(groupRole, 4001, 4001, []uint32{sharedGroup}, expected)
					run(otherRole, 4002, 4002, nil, expected)
				}
				checkReaders("original")
				run("agent", 65532, 65532, []uint32{sharedGroup}, "")
				var after unix.Stat_t
				if err := unix.Stat(target, &after); err != nil {
					t.Fatal(err)
				}
				expected := "replacement"
				if strings.HasSuffix(scenario, "failure") {
					expected = "original"
					if !reflect.DeepEqual(before, after) {
						// Reader probes may update atime, which is not access metadata.
						before.Atim = after.Atim
						if !reflect.DeepEqual(before, after) {
							t.Fatalf("failed write changed original inode: before=%+v after=%+v", before, after)
						}
					}
				} else if after.Uid != 65532 || after.Gid != uint32(gid) || after.Ino == before.Ino {
					t.Fatalf("expected atomic agent-owned replacement retaining GID %d: %+v", gid, after)
				}
				checkReaders(expected)
				run("game", 1000, 1000, []uint32{sharedGroup}, expected)
				entries, err := os.ReadDir(root)
				if err != nil || len(entries) != 1 || entries[0].Name() != "target.txt" {
					t.Fatalf("temporary file left behind: %v, %v", entries, err)
				}
			})
		}
	}
}

func runFileIdentityChild(t *testing.T, role string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Fatal("file operations must run unprivileged")
	}
	status, err := os.ReadFile("/proc/self/status")
	if err != nil || !strings.Contains(string(status), "CapEff:\t0000000000000000\n") {
		t.Fatalf("child has effective capabilities or cannot inspect them: %v", err)
	}
	root := os.Getenv("GAMEPLANE_FILE_ROOT")
	target := filepath.Join(root, "target.txt")
	if role == "agent" {
		if os.Getuid() != 65532 {
			t.Fatal("wrong agent UID")
		}
		h := &handler{root: root}
		scenario := os.Getenv("GAMEPLANE_FILE_SCENARIO")
		if scenario == "0600" {
			if _, err := os.ReadFile(target); !errors.Is(err, os.ErrPermission) {
				t.Fatalf("agent must initially lack content-read permission: %v", err)
			}
		}
		if scenario == "acl-failure" {
			h.preserveAccess = func(*os.File, fileAccess) error { return unix.EOPNOTSUPP }
		}
		wantFailure := strings.HasSuffix(scenario, "failure")
		if os.Getenv("GAMEPLANE_FILE_METHOD") == "write" {
			err := h.writeFile([]string{"target.txt"}, func(w io.Writer) error { _, err := io.WriteString(w, "replacement"); return err })
			if (wantFailure && !errors.Is(err, errPreserveAccess)) || (!wantFailure && err != nil) {
				t.Fatalf("writeFile: %v (want failure %v)", err, wantFailure)
			}
		} else {
			var body bytes.Buffer
			form := multipart.NewWriter(&body)
			part, err := form.CreateFormFile("file", "target.txt")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(part, "replacement"); err != nil {
				t.Fatal(err)
			}
			if err := form.Close(); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/files/upload?path=/", &body)
			req.Header.Set("Content-Type", form.FormDataContentType())
			response := httptest.NewRecorder()
			h.upload(response, req)
			wantStatus := http.StatusNoContent
			if wantFailure {
				wantStatus = http.StatusConflict
			}
			if response.Code != wantStatus {
				t.Fatalf("upload: status %d, body %s", response.Code, response.Body.String())
			}
		}
		return
	}
	content, err := os.ReadFile(target)
	if role == "denied" {
		if !errors.Is(err, os.ErrPermission) {
			t.Fatalf("unexpected read access: %q, %v", content, err)
		}
	} else if err != nil || string(content) != os.Getenv("GAMEPLANE_FILE_EXPECTED") {
		t.Fatalf("read: %q, %v", content, err)
	}
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_APPEND, 0)
	if role == "game" {
		if os.Getuid() != 1000 || err != nil {
			t.Fatalf("game UID %d cannot write: %v", os.Getuid(), err)
		}
		if _, err := fmt.Fprint(f, "\ngame still writes"); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(target)
		if err != nil || string(got) != os.Getenv("GAMEPLANE_FILE_EXPECTED")+"\ngame still writes" {
			t.Fatalf("game write not readable: %q, %v", got, err)
		}
	} else {
		if f != nil {
			_ = f.Close()
		}
		if !errors.Is(err, os.ErrPermission) {
			t.Fatalf("unexpected write access: %v", err)
		}
		if err := unix.Access(target, unix.X_OK); !errors.Is(err, unix.EACCES) {
			t.Fatalf("unexpected execute permission: %v", err)
		}
	}
}
