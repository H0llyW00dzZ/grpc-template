// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimiter defines the interface for checking if a request is allowed.
type RateLimiter interface {
	// Allow checks if the given key (e.g., IP address) is allowed to proceed.
	// It returns true if allowed, or false if the rate limit is exceeded.
	Allow(ctx context.Context, key string) (bool, error)
}

// MemoryRateLimiter is an in-memory implementation of RateLimiter
// using a token-bucket algorithm per peer key.
type MemoryRateLimiter struct {
	limiters    *peerLimiters
	rate        rate.Limit
	burst       int
	ttl         time.Duration
	cleanupStop chan struct{}
	stopOnce    sync.Once
}

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

// NewMemoryRateLimiter creates a new in-memory rate limiter and starts
// its background cleanup goroutine for stale entries.
// A rate of 0 or negative disables rate limiting.
func NewMemoryRateLimiter(rps float64, burst int, ttl time.Duration) *MemoryRateLimiter {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	m := &MemoryRateLimiter{
		limiters: &peerLimiters{
			limiters: make(map[string]*limiterEntry),
		},
		rate:        rate.Limit(rps),
		burst:       burst,
		ttl:         ttl,
		cleanupStop: make(chan struct{}),
	}
	if rps > 0 {
		go m.runCleanupLoop()
	}
	return m
}

// Allow checks the rate limit for the given key.
func (m *MemoryRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if m.rate <= 0 {
		return true, nil
	}
	limiter := m.limiters.getLimiter(key, m.rate, m.burst)
	return limiter.Allow(), nil
}

// Stop halts the background cleanup goroutine. It is safe to call
// multiple times and is concurrency-safe.
func (m *MemoryRateLimiter) Stop() {
	m.stopOnce.Do(func() {
		close(m.cleanupStop)
	})
}

func (m *MemoryRateLimiter) runCleanupLoop() {
	ticker := time.NewTicker(m.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.limiters.cleanup(m.ttl)
		case <-m.cleanupStop:
			return
		}
	}
}

// RateLimit returns a unary server interceptor that enforces per-peer
// rate limiting. It delegates to the currently configured RateLimiter.
func RateLimit() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		cfg := getConfig()
		if cfg.rateLimiter == nil {
			return handler(ctx, req)
		}

		key := peerKey(ctx, cfg.trustProxy)
		allowed, err := cfg.rateLimiter.Allow(ctx, key)
		if err != nil {
			return nil, status.Error(codes.Internal, "rate limiter internal error")
		}
		if !allowed {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}

// StreamRateLimit returns a stream server interceptor that enforces per-peer
// rate limiting. It delegates to the currently configured RateLimiter.
func StreamRateLimit() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		cfg := getConfig()
		if cfg.rateLimiter == nil {
			return handler(srv, ss)
		}

		key := peerKey(ss.Context(), cfg.trustProxy)
		allowed, err := cfg.rateLimiter.Allow(ss.Context(), key)
		if err != nil {
			return status.Error(codes.Internal, "rate limiter internal error")
		}
		if !allowed {
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

// peerKey extracts the client IP from the gRPC peer information.
// When trustProxy is true it first checks common proxy headers
// (x-forwarded-for, x-real-ip) in the gRPC metadata.
// Otherwise, it falls back to the direct hardware peer connection.
//
// The caller must pass the trustProxy value from its own config
// snapshot so that a single consistent configuration generation is
// used for the entire request.
func peerKey(ctx context.Context, trustProxy bool) string {
	if trustProxy {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ips := md.Get("x-forwarded-for"); len(ips) > 0 && ips[0] != "" {
				// x-forwarded-for can be a comma-separated list of IPs.
				// The true client is the first IP.
				clientIP := strings.Split(ips[0], ",")[0]
				return strings.TrimSpace(clientIP)
			}
			if ips := md.Get("x-real-ip"); len(ips) > 0 && ips[0] != "" {
				return strings.TrimSpace(ips[0])
			}
		}
	}

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
