// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"sync"
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

// CleanupLimiters exports cleanup for testing.
func CleanupLimiters(ttl time.Duration) {
	globalLimiters.cleanup(ttl)
}

// LimiterCount returns the number of active peer limiters for testing.
func LimiterCount() int {
	globalLimiters.mu.Lock()
	defer globalLimiters.mu.Unlock()
	return len(globalLimiters.limiters)
}

// SetLimiterLastSeen sets the lastSeen time for a specific peer key
// to enable TTL-based cleanup testing.
func SetLimiterLastSeen(key string, t time.Time) {
	globalLimiters.mu.Lock()
	defer globalLimiters.mu.Unlock()
	if entry, ok := globalLimiters.limiters[key]; ok {
		entry.lastSeen = t
	}
}

// ResetCleanupOnce resets the sync.Once guard so startCleanup can be
// invoked again during testing.
func ResetCleanupOnce() {
	cleanupOnce = sync.Once{}
}

// StartCleanup exports startCleanup for testing.
func StartCleanup(ttl time.Duration) {
	startCleanup(ttl)
}

// StopCleanup signals the cleanup goroutine to stop.
func StopCleanup() {
	if cleanupStop != nil {
		close(cleanupStop)
	}
}

// RunCleanupLoop exports runCleanupLoop for direct testing.
func RunCleanupLoop(ttl time.Duration, stop <-chan struct{}) {
	runCleanupLoop(ttl, stop)
}
