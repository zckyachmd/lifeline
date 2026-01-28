package ratelimit

import (
	"sync"
	"time"
)

// Limiter provides per-user sliding window limiting.
type Limiter struct {
	window   time.Duration
	maxCalls int
	mu       sync.Mutex
	users    map[int64][]time.Time
}

// New creates a limiter with maxCalls per window.
func New(maxCalls int, window time.Duration) *Limiter {
	return &Limiter{
		window:   window,
		maxCalls: maxCalls,
		users:    make(map[int64][]time.Time),
	}
}

// Allow checks and records a new request.
func (l *Limiter) Allow(user int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	windowStart := now.Add(-l.window)

	times := l.users[user]
	// drop old timestamps
	pruned := times[:0]
	for _, t := range times {
		if t.After(windowStart) {
			pruned = append(pruned, t)
		}
	}
	l.users[user] = pruned

	if len(pruned) >= l.maxCalls {
		return false
	}
	l.users[user] = append(pruned, now)
	return true
}
