package main

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"
)

// Per-source limiter bounds (research R6). All limiter state is in memory
// only; source addresses are never written to the store or the logs (FR-014).
const (
	// limiterIdleTTL is how long an idle token bucket is kept before it is
	// purged. An idle bucket has refilled completely, so purging loses nothing.
	limiterIdleTTL = 10 * time.Minute
	// limiterSweepEvery is how often the lazy idle sweep runs.
	limiterSweepEvery = time.Minute
	// limiterMaxSources bounds how many source prefixes one limiter tracks. When
	// the map is full and a new key arrives, the request is accounted against a
	// shared overflow entry that shares the same limits as a single source,
	// avoiding unbounded memory growth from a flood of distinct addresses.
	limiterMaxSources = 100_000
)

// overflowKey is a sentinel prefix used when a limiter is at capacity and a
// new address arrives. All overflow traffic shares one rate budget.
var overflowKey = netip.MustParsePrefix("192.0.2.0/24")

// sourceKey derives a rate-limit key from an address. IPv4 addresses and
// IPv4-mapped IPv6 addresses (after Unmap) give a /32. Other IPv6 addresses
// give a /64. Invalid addresses give a fixed sentinel. This prevents IPv6
// address rotation bypasses while keeping routing subnets together.
func sourceKey(a netip.Addr) netip.Prefix {
	if !a.IsValid() {
		return overflowKey
	}
	a = a.Unmap()
	if a.Is4() {
		return netip.PrefixFrom(a, 32)
	}
	p, err := a.WithZone("").Prefix(64)
	if err != nil {
		return overflowKey
	}
	return p
}

// dayString formats t as a UTC calendar day, YYYY-MM-DD.
func dayString(t time.Time) string {
	return t.UTC().Format(time.DateOnly)
}

// dailyLimiter counts accepted reports per source per UTC day. The counts
// are reset when the UTC day changes. A limit of 0 means unlimited and keeps
// no state.
type dailyLimiter struct {
	mu     sync.Mutex
	limit  int
	max    int
	day    string
	counts map[netip.Prefix]int
}

func newDailyLimiter(limit int) *dailyLimiter {
	return &dailyLimiter{limit: limit, max: limiterMaxSources, counts: make(map[netip.Prefix]int)}
}

// take records one accepted report from src at now. It returns false, and
// records nothing, when src has already used its limit for the UTC day of now.
func (l *dailyLimiter) take(src netip.Addr, now time.Time) bool {
	if l.limit <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if day := dayString(now); day != l.day {
		l.day = day
		l.counts = make(map[netip.Prefix]int)
	}
	key := sourceKey(src)
	n, tracked := l.counts[key]
	if !tracked && len(l.counts) >= l.max {
		key = overflowKey
		n, _ = l.counts[key]
	}
	if n >= l.limit {
		return false
	}
	l.counts[key] = n + 1
	return true
}

// bucketLimiter is a generic per-source token bucket.
type bucketLimiter struct {
	mu        sync.Mutex
	rate      float64 // tokens added per second
	burst     float64
	max       int
	day       string
	lastSweep time.Time
	buckets   map[netip.Prefix]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

// newBucketLimiter returns a limiter that refills perMinute tokens per
// minute up to burst. A new source starts with a full bucket.
func newBucketLimiter(perMinute, burst int) *bucketLimiter {
	return &bucketLimiter{
		rate:    float64(perMinute) / 60,
		burst:   float64(burst),
		max:     limiterMaxSources,
		buckets: make(map[netip.Prefix]*bucket),
	}
}

// allow takes one token for src at now and reports whether one was left.
func (l *bucketLimiter) allow(src netip.Addr, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep(now)
	key := sourceKey(src)
	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= l.max {
			key = overflowKey
			b = l.buckets[key]
			if b == nil {
				b = &bucket{tokens: l.burst, last: now}
				l.buckets[key] = b
			}
		} else {
			b = &bucket{tokens: l.burst, last: now}
			l.buckets[key] = b
		}
	}
	if elapsed := now.Sub(b.last).Seconds(); elapsed > 0 {
		b.tokens = min(l.burst, b.tokens+elapsed*l.rate)
		b.last = now
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// sweep resets every bucket at UTC midnight and, at most once a minute,
// purges buckets idle for longer than limiterIdleTTL. The caller holds mu.
func (l *bucketLimiter) sweep(now time.Time) {
	if day := dayString(now); day != l.day {
		l.day = day
		l.buckets = make(map[netip.Prefix]*bucket)
		l.lastSweep = now
		return
	}
	if now.Sub(l.lastSweep) < limiterSweepEvery {
		return
	}
	l.lastSweep = now
	for src, b := range l.buckets {
		if now.Sub(b.last) >= limiterIdleTTL {
			delete(l.buckets, src)
		}
	}
}

// sourceIP returns the address a request is attributed to: the TCP peer,
// unless the peer is inside trusted, in which case it is the right-most
// X-Forwarded-For entry that is not itself trusted. An unparsable
// X-Forwarded-For falls back to the peer. An unparsable peer yields the zero
// Addr, which all such requests share.
func sourceIP(r *http.Request, trusted []netip.Prefix) netip.Addr {
	peer := peerAddr(r.RemoteAddr)
	if !peer.IsValid() || !inPrefixes(trusted, peer) {
		return peer
	}
	parts := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(parts) - 1; i >= 0; i-- {
		raw := strings.TrimSpace(parts[i])
		if raw == "" {
			continue
		}
		a, err := netip.ParseAddr(raw)
		if err != nil {
			return peer
		}
		a = a.Unmap().WithZone("")
		if !inPrefixes(trusted, a) {
			return a
		}
	}
	return peer
}

// peerAddr parses a net/http RemoteAddr ("host:port").
func peerAddr(remote string) netip.Addr {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		host = remote
	}
	a, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return a.Unmap().WithZone("")
}

func inPrefixes(prefixes []netip.Prefix, a netip.Addr) bool {
	for _, p := range prefixes {
		if p.Contains(a) {
			return true
		}
	}
	return false
}
