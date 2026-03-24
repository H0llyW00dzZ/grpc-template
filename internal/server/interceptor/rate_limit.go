// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// peerLimiters manages per-peer rate limiters with automatic cleanup
// of stale entries to prevent memory leaks.
type peerLimiters struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
}

// limiterEntry holds a rate limiter and the time it was last accessed.
type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// globalLimiters is the package-level limiter store, initialised lazily
// by the first call to [RateLimit] or [StreamRateLimit].
var globalLimiters = &peerLimiters{
	limiters: make(map[string]*limiterEntry),
}

// cleanupOnce ensures the background cleanup goroutine is started exactly once.
var cleanupOnce sync.Once

// RateLimit returns a unary server interceptor that enforces per-peer
// rate limiting using a token-bucket algorithm.
//
// Each unique client IP receives its own [rate.Limiter] with the rate
// and burst values configured via [WithRateLimit]. If the caller
// exceeds its allowance the interceptor returns [codes.ResourceExhausted].
//
// Stale limiter entries are cleaned up automatically in the background.
func RateLimit() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		cfg := defaultConfig
		if cfg.rateLimit <= 0 {
			return handler(ctx, req)
		}

		key := peerKey(ctx)
		limiter := globalLimiters.getLimiter(key, cfg.rateLimit, cfg.rateBurst)

		startCleanup(cfg.rateLimitTTL)

		if !limiter.Allow() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}

// StreamRateLimit returns a stream server interceptor that enforces per-peer
// rate limiting using a token-bucket algorithm.
//
// Each unique client IP receives its own [rate.Limiter] with the rate
// and burst values configured via [WithRateLimit]. If the caller
// exceeds its allowance the interceptor returns [codes.ResourceExhausted].
func StreamRateLimit() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		cfg := defaultConfig
		if cfg.rateLimit <= 0 {
			return handler(srv, ss)
		}

		key := peerKey(ss.Context())
		limiter := globalLimiters.getLimiter(key, cfg.rateLimit, cfg.rateBurst)

		startCleanup(cfg.rateLimitTTL)

		if !limiter.Allow() {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(srv, ss)
	}
}

// getLimiter returns the rate limiter for the given key, creating a new
// one if none exists. It also updates the last-seen timestamp.
func (p *peerLimiters) getLimiter(key string, r rate.Limit, burst int) *rate.Limiter {
	p.mu.Lock()
	defer p.mu.Unlock()

	if entry, ok := p.limiters[key]; ok {
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	limiter := rate.NewLimiter(r, burst)
	p.limiters[key] = &limiterEntry{
		limiter:  limiter,
		lastSeen: time.Now(),
	}
	return limiter
}

// cleanup removes limiter entries that have not been accessed within
// the given TTL duration.
func (p *peerLimiters) cleanup(ttl time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cutoff := time.Now().Add(-ttl)
	for key, entry := range p.limiters {
		if entry.lastSeen.Before(cutoff) {
			delete(p.limiters, key)
		}
	}
}

// startCleanup ensures the background cleanup goroutine runs exactly once.
func startCleanup(ttl time.Duration) {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	cleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(ttl)
			defer ticker.Stop()
			for range ticker.C {
				globalLimiters.cleanup(ttl)
			}
		}()
	})
}

// peerKey extracts the client IP from the gRPC peer information.
// Falls back to "unknown" if peer info is unavailable.
func peerKey(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return "unknown"
	}

	// Strip the port to group all connections from the same IP.
	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return p.Addr.String()
	}
	return host
}
