package kube

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestPodLogsKeepsTerminalFailureBeyondFormerReadLimit(t *testing.T) {
	// Fewer than the requested 200 lines, but more than the former 4 MiB
	// read limit. The last line is the failure the caller needs to diagnose.
	logs := strings.Repeat(strings.Repeat("x", 32<<10)+"\n", 160) + "fatal: terminal failure at end of logs\n"
	clientset := k8sfake.NewSimpleClientset()
	clientset.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.GetSubresource() != "log" {
			return false, nil, nil
		}
		return true, &runtime.Unknown{Raw: []byte(logs)}, nil
	})
	c := &Client{typed: clientset}
	got, err := c.PodLogs(context.Background(), "games", "crashed", "game", 200, true)
	if err != nil {
		t.Fatal(err)
	}
	want := truncatedLogNotice + logs[len(logs)-maxLogBytes:]
	if got != want {
		t.Fatalf("log tail does not match newest bytes: got %d bytes, want %d", len(got), len(want))
	}
}

func TestReadLogTailBoundariesAndChunking(t *testing.T) {
	for _, size := range []int{0, 1, maxLogBytes - 1, maxLogBytes, maxLogBytes + 1, maxLogBytes*21 + 127} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			input := strings.Repeat("0123456789abcdef", (size+15)/16)[:size]
			want := input[max(0, size-maxLogBytes):]
			for _, chunked := range []bool{false, true} {
				var reader io.Reader = strings.NewReader(input)
				if chunked {
					reader = iotest.HalfReader(reader)
				}
				got, truncated, err := readLogTail(reader)
				if err != nil || string(got) != want || truncated != (size > maxLogBytes) {
					t.Fatalf("chunked=%v: got %d bytes, truncated=%v, err=%v", chunked, len(got), truncated, err)
				}
			}
		})
	}
}

func TestReadLogTailFailsOnIncompleteStream(t *testing.T) {
	for _, failure := range []error{io.ErrUnexpectedEOF, context.Canceled} {
		stream := io.MultiReader(strings.NewReader(strings.Repeat("old logs\n", maxLogBytes)), iotest.ErrReader(failure))
		got, truncated, err := readLogTail(stream)
		if !errors.Is(err, failure) || len(got) != 0 || truncated {
			t.Fatalf("incomplete stream returned as logs: %d bytes, truncated=%v, err=%v", len(got), truncated, err)
		}
	}
}

func TestLogTailRetentionStaysBounded(t *testing.T) {
	tail := logTail{buf: make([]byte, maxLogBytes)}
	chunk := []byte(strings.Repeat("line\n", 10001))
	for i := 0; i < 200; i++ {
		n, err := tail.Write(chunk)
		if n != len(chunk) || err != nil {
			t.Fatalf("Write = %d, %v", n, err)
		}
		if len(tail.buf) != maxLogBytes || cap(tail.buf) != maxLogBytes || tail.size > maxLogBytes {
			t.Fatal("retained buffer grew beyond the log output limit")
		}
	}
}
