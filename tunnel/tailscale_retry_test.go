package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestTailnetRetryHealthRequiresContinuousConfirmation(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"short session", "healthy interval", "offline interruption", "stale gap"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			backoff := &exponentialBackoff{}
			for _, want := range []time.Duration{1, 2, 4, 8, 16, 32, 64, 128, 256, 300, 300} {
				if got := backoff.next(); got != want*time.Second {
					t.Fatalf("failure delay=%v, want %v", got, want*time.Second)
				}
			}
			history := &tailnetRetryHealth{}
			start := time.Unix(100, 0)
			observe := func(seconds int, healthy bool) {
				history.observe(start.Add(time.Duration(seconds)*time.Second), healthy, backoff)
			}
			for seconds := 0; seconds <= 58; seconds += 2 {
				observe(seconds, true)
			}
			if got := backoff.next(); got != 300*time.Second {
				t.Fatalf("short healthy session reset delay: %v", got)
			}
			switch scenario {
			case "short session":
				observe(59, false)
				if got := backoff.next(); got != 300*time.Second {
					t.Fatalf("session exit reset delay: %v", got)
				}
				return
			case "healthy interval":
				observe(60, true)
			case "offline interruption":
				observe(60, false)
				for seconds := 62; seconds <= 120; seconds += 2 {
					observe(seconds, true)
				}
				if got := backoff.next(); got != 300*time.Second {
					t.Fatalf("offline interruption counted as healthy time: %v", got)
				}
				observe(122, true)
			case "stale gap":
				// Exactly ten seconds without an observation breaks continuity.
				for seconds := 68; seconds <= 126; seconds += 2 {
					observe(seconds, true)
				}
				if got := backoff.next(); got != 300*time.Second {
					t.Fatalf("stale observation gap counted as healthy time: %v", got)
				}
				observe(128, true)
			}
			observe(130, false)
			if got := backoff.next(); got != time.Second {
				t.Fatalf("confirmed healthy interval did not reset: %v", got)
			}
			if got := backoff.next(); got != 2*time.Second {
				t.Fatalf("subsequent failure did not escalate: %v", got)
			}
		})
	}
}

func TestTailnetConfirmedObservationsResetRetryEscalation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	node := &fakeTailnetNode{status: onlineTailnet()}
	backoff := &exponentialBackoff{}
	for i := 0; i < 12; i++ {
		backoff.next()
	}
	history := &tailnetRetryHealth{}
	var ready atomic.Bool
	confirmed := 0
	start := time.Unix(100, 0)
	err := serveTailnetObserved(ctx, node, nil, []forwardPort{{network: "tcp", port: "25565", target: "127.0.0.1:1"}}, &ready, time.Millisecond, func(healthy bool) {
		history.observe(start.Add(time.Duration(confirmed)*2*time.Second), healthy, backoff)
		if healthy {
			confirmed++
			if confirmed == 31 {
				cancel()
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if confirmed < 31 || ready.Load() {
		t.Fatalf("confirmed=%d ready=%v", confirmed, ready.Load())
	}
	if got := backoff.next(); got != time.Second {
		t.Fatalf("confirmed tailnet health did not reset backoff: %v", got)
	}
	for _, listener := range node.listeners {
		if _, err := listener.Accept(); err == nil {
			t.Fatal("listener survives cancellation")
		}
	}
}

func TestTailnetFailedAttemptsCannotResetRetryEscalation(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"authentication", "denied tag", "listener"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			node := &fakeTailnetNode{status: onlineTailnet()}
			tags := []string{"tag:game"}
			switch scenario {
			case "authentication":
				node.upErr = errors.New("unauthorized")
			case "denied tag":
				tags = []string{"tag:ungranted"}
			case "listener":
				node.listenErr = true
			}
			backoff := &exponentialBackoff{}
			backoff.next()
			history := &tailnetRetryHealth{}
			var ready atomic.Bool
			confirmed := false
			err := serveTailnetObserved(t.Context(), node, tags, []forwardPort{{network: "udp", port: "19132", target: "127.0.0.1:1"}}, &ready, time.Millisecond, func(healthy bool) {
				confirmed = confirmed || healthy
				history.observe(time.Unix(100, 0), healthy, backoff)
			})
			if err == nil || ready.Load() || confirmed {
				t.Fatalf("failed attempt reported health: err=%v ready=%v confirmed=%v", err, ready.Load(), confirmed)
			}
			if got := backoff.next(); got != 2*time.Second {
				t.Fatalf("failed attempt reset delay: %v", got)
			}
		})
	}
}
