package main

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
)

func fakeFS(files map[string]string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		if v, ok := files[path]; ok {
			return []byte(v), nil
		}
		return nil, errors.New("not found")
	}
}

func noEnv(string) (string, bool) { return "", false }

func TestCgroupMemoryLimit(t *testing.T) {
	const mib = int64(1) << 20
	cases := []struct {
		name   string
		files  map[string]string
		want   int64
		wantOK bool
	}{
		{"v2 limit", map[string]string{cgroupV2MemoryMax: "268435456\n"}, 256 * mib, true},
		{"v2 unlimited", map[string]string{cgroupV2MemoryMax: "max\n"}, 0, false},
		{"v1 limit", map[string]string{cgroupV1MemoryLimit: "134217728\n"}, 128 * mib, true},
		{"v1 unlimited", map[string]string{cgroupV1MemoryLimit: "9223372036854771712\n"}, 0, false},
		{"v2 max falls through to v1", map[string]string{cgroupV2MemoryMax: "max", cgroupV1MemoryLimit: "134217728"}, 128 * mib, true},
		{"garbage", map[string]string{cgroupV2MemoryMax: "lots"}, 0, false},
		{"zero", map[string]string{cgroupV2MemoryMax: "0"}, 0, false},
		{"no files", map[string]string{}, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := cgroupMemoryLimit(fakeFS(tc.files))
			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("cgroupMemoryLimit = (%d, %v), want (%d, %v)", got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestSoftMemoryLimit_Is80PercentOfCgroupLimit(t *testing.T) {
	read := fakeFS(map[string]string{cgroupV2MemoryMax: "268435456"}) // 256 MiB
	got, ok := softMemoryLimit(noEnv, read)
	if !ok {
		t.Fatal("want a limit")
	}
	want := int64(268435456) / 100 * 80
	if got != want {
		t.Fatalf("limit = %d, want %d", got, want)
	}
	// 12 concurrent logins with one argon slot: 64 MiB live + ~50 MiB
	// baseline must fit under the soft limit.
	if got < (64+50)<<20 {
		t.Fatalf("soft limit %d leaves no room for one argon2 block plus baseline", got)
	}
}

func TestSoftMemoryLimit_ExplicitGOMEMLIMITWins(t *testing.T) {
	read := fakeFS(map[string]string{cgroupV2MemoryMax: "268435456"})
	lookup := func(k string) (string, bool) { return "off", k == "GOMEMLIMIT" }
	if got, ok := softMemoryLimit(lookup, read); ok {
		t.Fatalf("GOMEMLIMIT set: want no override, got %d", got)
	}
}

func TestSoftMemoryLimit_NoCgroupLimit(t *testing.T) {
	if got, ok := softMemoryLimit(noEnv, fakeFS(nil)); ok {
		t.Fatalf("no cgroup limit: want no override, got %d", got)
	}
}

func TestApplyMemoryLimit_GOMEMLIMITSetIsNoop(t *testing.T) {
	t.Setenv("GOMEMLIMIT", "1GiB")
	if _, set := os.LookupEnv("GOMEMLIMIT"); !set {
		t.Fatal("setup: GOMEMLIMIT not set")
	}
	applyMemoryLimit(slog.New(slog.NewTextHandler(io.Discard, nil))) // must not change the runtime limit or panic
}
