// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"time"
)

// PeerKey is exported for testing. It calls the unexported peerKey function.
func PeerKey(ctx interface{ Value(any) any }) string {
	// Type assert to context.Context by using the concrete function.
	return peerKey(ctx.(interface {
		Deadline() (time.Time, bool)
		Done() <-chan struct{}
		Err() error
		Value(any) any
	}))
}

// ActiveMemoryLimiter gets the currently configured MemoryRateLimiter for testing.
func ActiveMemoryLimiter() *MemoryRateLimiter {
	cfg := getConfig()
	if m, ok := cfg.rateLimiter.(*MemoryRateLimiter); ok {
		return m
	}
	return nil
}

// CleanupLimiters exports cleanup for testing on the active memory limiter.
func CleanupLimiters(ttl time.Duration) {
	if m := ActiveMemoryLimiter(); m != nil {
		m.limiters.cleanup(ttl)
	}
}

// LimiterCount returns the number of active peer limiters for testing.
func LimiterCount() int {
	m := ActiveMemoryLimiter()
	if m == nil {
		return 0
	}
	m.limiters.mu.Lock()
	defer m.limiters.mu.Unlock()
	return len(m.limiters.limiters)
}

// SetLimiterLastSeen sets the lastSeen time for a specific peer key
// to enable TTL-based cleanup testing.
func SetLimiterLastSeen(key string, t time.Time) {
	m := ActiveMemoryLimiter()
	if m == nil {
		return
	}
	m.limiters.mu.Lock()
	defer m.limiters.mu.Unlock()
	if entry, ok := m.limiters.limiters[key]; ok {
		entry.lastSeen = t
	}
}

// StopCleanup signals the cleanup goroutine to stop on the active memory limiter.
func StopCleanup() {
	if m := ActiveMemoryLimiter(); m != nil {
		m.Stop()
	}
}
