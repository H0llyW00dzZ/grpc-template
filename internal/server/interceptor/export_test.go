// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"
	"time"
)

// PeerKey is exported for testing. It calls the unexported peerKey function
// with the given trustProxy setting.
func PeerKey(ctx context.Context, trustProxy bool) string {
	return peerKey(ctx, trustProxy)
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

// StopCleanup signals the cleanup goroutine to stop on the active memory
// limiter, if one is configured. Safe to call when no limiter is active.
func StopCleanup() {
	if m := ActiveMemoryLimiter(); m != nil {
		m.Stop()
	}
}

// ExtractBearerToken is exported for benchmarking. It calls the unexported
// extractBearerToken function.
func ExtractBearerToken(ctx context.Context) (string, error) {
	return extractBearerToken(ctx)
}

// ResetConfig resets the package-level configuration to defaults.
// It stops any previously configured rate limiter to prevent cleanup
// goroutine leaks between tests.
// This is only available in tests.
func ResetConfig() {
	configMu.Lock()
	defer configMu.Unlock()
	stopPreviousLimiter(defaultConfig.rateLimiter)
	defaultConfig = &config{
		excludedMethods: make(map[string]struct{}),
		demotedMethods:  defaultDemotedMethods(),
	}
}
