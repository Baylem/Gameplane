package kube

import "io"

// readLogTail consumes the complete non-following Kubernetes response. Stopping
// after a byte budget would return the middle of a large tail as its newest
// output. The ring retains a fixed amount regardless of response size; stream
// errors (including request cancellation) are never presented as a complete tail.
func readLogTail(r io.Reader) ([]byte, bool, error) {
	tail := logTail{buf: make([]byte, maxLogBytes)}
	if _, err := io.Copy(&tail, r); err != nil {
		return nil, false, err
	}
	data := make([]byte, tail.size)
	if tail.size < len(tail.buf) {
		copy(data, tail.buf[:tail.size])
	} else {
		n := copy(data, tail.buf[tail.next:])
		copy(data[n:], tail.buf[:tail.next])
	}
	return data, tail.truncated, nil
}

type logTail struct {
	buf       []byte
	next      int
	size      int
	truncated bool
}

func (t *logTail) Write(p []byte) (int, error) {
	n := len(p)
	if n > len(t.buf)-t.size {
		t.truncated = true
	}
	if n >= len(t.buf) {
		copy(t.buf, p[n-len(t.buf):])
		t.next = 0
		t.size = len(t.buf)
		return n, nil
	}
	first := copy(t.buf[t.next:], p)
	copy(t.buf, p[first:])
	t.next = (t.next + n) % len(t.buf)
	t.size = min(t.size+n, len(t.buf))
	return n, nil
}
