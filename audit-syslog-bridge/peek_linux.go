//go:build linux

package main

import (
	"crypto/tls"
	"net"
	"syscall"
)

// peek performs a non-blocking socket peek to determine connection state.
// On Linux, uses MSG_PEEK|MSG_DONTWAIT to check for pending data or a close
// without blocking.
func peek(c net.Conn) peekState {
	// Unwrap *tls.Conn to get the underlying net.Conn.
	tc, ok := c.(*tls.Conn)
	if ok {
		c = tc.NetConn()
	}

	// Get syscall.Conn to perform raw syscall reads.
	sc, ok := c.(syscall.Conn)
	if !ok {
		return peekUnknown
	}

	rc, err := sc.SyscallConn()
	if err != nil {
		return peekUnknown
	}

	var (
		buf  [1]byte
		n    int
		err2 error
	)
	rerr := rc.Read(func(fd uintptr) bool {
		n, _, err2 = syscall.Recvfrom(int(fd), buf[:1], syscall.MSG_PEEK|syscall.MSG_DONTWAIT)
		return true
	})

	if rerr != nil {
		return peekUnknown
	}

	if err2 != nil {
		switch err2 {
		case syscall.EAGAIN:
			// No data pending; connection is open. (EWOULDBLOCK is an alias for EAGAIN on Linux)
			return peekEmpty
		case syscall.EINTR:
			// Interrupted system call; inconclusive.
			return peekUnknown
		default:
			// Any other error (ECONNRESET, EPIPE, etc.) indicates closure.
			return peekClosed
		}
	}

	// n == 0 means a FIN was received (connection closed).
	if n == 0 {
		return peekClosed
	}

	// n > 0 means data is pending.
	return peekPending
}
