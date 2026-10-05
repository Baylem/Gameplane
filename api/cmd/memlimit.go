package main

import (
	"log/slog"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

const (
	// memLimitPercent is the share of the container's memory limit given to
	// the Go runtime as a soft limit. The rest is headroom for memory the Go
	// heap accounting does not cover but the cgroup does (kernel/socket
	// buffers, thread stacks, mapped files).
	memLimitPercent = 80

	cgroupV2MemoryMax   = "/sys/fs/cgroup/memory.max"
	cgroupV1MemoryLimit = "/sys/fs/cgroup/memory/memory.limit_in_bytes"

	// cgroup v1 reports "no limit" as a value close to MaxInt64 (page
	// aligned); anything above 1 PiB is treated as unlimited.
	cgroupUnlimitedThreshold = int64(1) << 50
)

// cgroupMemoryLimit returns the container memory limit in bytes from cgroup
// v2 (memory.max) or v1 (memory.limit_in_bytes), reading files through read
// so tests can fake them. ok is false when no finite limit is configured.
func cgroupMemoryLimit(read func(string) ([]byte, error)) (limit int64, ok bool) {
	for _, path := range []string{cgroupV2MemoryMax, cgroupV1MemoryLimit} {
		raw, err := read(path)
		if err != nil {
			continue
		}
		s := strings.TrimSpace(string(raw))
		if s == "max" {
			continue
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n <= 0 || n >= cgroupUnlimitedThreshold {
			continue
		}
		return n, true
	}
	return 0, false
}

// softMemoryLimit computes the Go soft memory limit to apply. ok is false
// when GOMEMLIMIT is set (the operator's explicit choice wins, including
// GOMEMLIMIT=off) or the container has no finite cgroup memory limit.
func softMemoryLimit(lookupEnv func(string) (string, bool), read func(string) ([]byte, error)) (limit int64, ok bool) {
	if _, set := lookupEnv("GOMEMLIMIT"); set {
		return 0, false
	}
	cg, ok := cgroupMemoryLimit(read)
	if !ok {
		return 0, false
	}
	return cg / 100 * memLimitPercent, true
}

// applyMemoryLimit sets the Go runtime soft memory limit from the cgroup limit
// when GOMEMLIMIT is unset. Without it the default GC (GOGC=100) retains freed
// 64 MiB argon2 blocks until the heap doubles, which OOM-kills the 256 MiB API
// container under a burst of concurrent logins even though the argon2
// semaphore bounds live blocks.
func applyMemoryLimit(logger *slog.Logger) {
	limit, ok := softMemoryLimit(os.LookupEnv, os.ReadFile)
	if !ok {
		return
	}
	debug.SetMemoryLimit(limit)
	logger.Info("go memory limit set from cgroup", "bytes", limit, "percent", memLimitPercent)
}
