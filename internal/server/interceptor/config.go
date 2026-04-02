// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"
	"sync"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
)

// AuthFunc is a user-provided function that validates a token string
// and returns an enriched context (e.g., with user claims) or an error.
// The interceptor extracts the bearer token from the "authorization"
// metadata key and passes it to this function.
type AuthFunc func(ctx context.Context, token string) (context.Context, error)

// config holds shared configuration for all interceptors in this package.
type config struct {
	logger          logging.Handler
	authFunc        AuthFunc
	excludedMethods map[string]struct{}
	demotedMethods  map[string]struct{} // methods demoted to Debug level on Canceled
	rateLimiter     RateLimiter
	trustProxy      bool
}

// configMu guards all reads and writes to defaultConfig.
var configMu sync.RWMutex

// defaultConfig is the package-level configuration used by all interceptors.
var defaultConfig = &config{
	excludedMethods: make(map[string]struct{}),
	demotedMethods:  defaultDemotedMethods(),
}

// getConfig returns a snapshot of the current package-level configuration
// under a read lock. The returned struct is safe to use without holding
// the lock because its value fields are copied; however the map and
// interface fields still alias the originals (which are only written at
// init time via [Configure]).
func getConfig() config {
	configMu.RLock()
	defer configMu.RUnlock()
	return *defaultConfig
}

// isDemoted reports whether the given method should be demoted to Debug
// level when the RPC completes with [codes.Canceled].
func (c config) isDemoted(method string) bool {
	_, ok := c.demotedMethods[method]
	return ok
}

// Option configures the interceptor package.
type Option func(*config)

// WithLogger sets the logger used by interceptors that perform logging
// (e.g., [Logging], [Recovery]).
//
// If not set, interceptors fall back to [logging.Default].
func WithLogger(l logging.Handler) Option {
	return func(c *config) {
		c.logger = l
	}
}

// WithAuthFunc sets the authentication function used by [Auth] and [StreamAuth].
// The function receives the bearer token extracted from the "authorization"
// metadata and should return an enriched context or an error.
func WithAuthFunc(fn AuthFunc) Option {
	return func(c *config) {
		c.authFunc = fn
	}
}

// WithExcludedMethods configures [Auth] and [StreamAuth] to skip
// authentication for the given fully-qualified gRPC method names.
//
//	interceptor.Configure(
//	    interceptor.WithAuthFunc(myAuthFunc),
//	    interceptor.WithExcludedMethods(
//	        "/grpc.health.v1.Health/Check",
//	        "/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
//	    ),
//	)
func WithExcludedMethods(methods ...string) Option {
	return func(c *config) {
		for _, m := range methods {
			if m != "" {
				c.excludedMethods[m] = struct{}{}
			}
		}
	}
}

// WithDemotedMethods configures [Logging] and [StreamLogging] to demote
// matching RPC errors from Error to Debug level when the gRPC status
// code is [codes.Canceled]. This is useful for methods like
// ServerReflection where client-initiated cancellation is expected.
//
// By default, both gRPC reflection v1 and v1alpha methods are demoted.
// Calling this option adds to the existing set; it does not replace it.
//
//	interceptor.Configure(
//	    interceptor.WithDemotedMethods(
//	        "/myapp.v1.LongPoll/Watch",
//	    ),
//	)
func WithDemotedMethods(methods ...string) Option {
	return func(c *config) {
		for _, m := range methods {
			if m != "" {
				c.demotedMethods[m] = struct{}{}
			}
		}
	}
}

// defaultDemotedMethods returns the built-in set of methods that are
// demoted to Debug level when cancelled. These are the standard gRPC
// reflection endpoints whose cancellation is expected behaviour.
func defaultDemotedMethods() map[string]struct{} {
	return map[string]struct{}{
		"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo":      {},
		"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo": {},
	}
}

// Configure applies the given options to the package-level interceptor
// configuration. Call this once during application startup to share
// settings (such as a logger) across all interceptors.
//
//	interceptor.Configure(
//	    interceptor.WithLogger(myLogger),
//	    interceptor.WithAuthFunc(myAuthFunc),
//	    interceptor.WithExcludedMethods("/grpc.health.v1.Health/Check"),
//	)
//
// When using the [server] package, [server.WithLogger] calls this
// automatically—no manual configuration is needed.
func Configure(opts ...Option) {
	configMu.Lock()
	defer configMu.Unlock()
	for _, opt := range opts {
		opt(defaultConfig)
	}
}

// WithRateLimiter sets a custom rate limiter implementation.
// When set, this overrides any limiter configured via [WithRateLimit],
// since both options write to the same config field.
//
// If the previous rate limiter implements a Stop method (e.g.,
// [MemoryRateLimiter]), it is stopped automatically to prevent
// background goroutine leaks.
//
//	interceptor.Configure(
//	    interceptor.WithRateLimiter(interceptor.NewMemoryRateLimiter(100, 200, 10*time.Minute)),
//	)
func WithRateLimiter(l RateLimiter) Option {
	return func(c *config) {
		stopPreviousLimiter(c.rateLimiter)
		c.rateLimiter = l
	}
}

// WithRateLimit is a convenience option that configures the default
// in-memory rate limiter with a 10-minute TTL.
// A rate of 0 or negative disables rate limiting.
//
// If [WithRateLimiter] is also used, whichever is applied last takes effect.
// The previous limiter (if any) is stopped automatically.
//
//	interceptor.Configure(
//	    interceptor.WithRateLimit(100, 200), // 100 req/s, burst up to 200
//	)
func WithRateLimit(rps float64, burst int) Option {
	return func(c *config) {
		stopPreviousLimiter(c.rateLimiter)
		c.rateLimiter = NewMemoryRateLimiter(rps, burst, 10*time.Minute)
	}
}

// stoppable is an optional interface that rate limiters can implement
// to allow cleanup of background resources when replaced.
type stoppable interface {
	Stop()
}

// stopPreviousLimiter stops the given rate limiter's background
// resources if it implements the [stoppable] interface.
func stopPreviousLimiter(rl RateLimiter) {
	if s, ok := rl.(stoppable); ok {
		s.Stop()
	}
}

// WithTrustProxy configures the rate limiter to trust the X-Forwarded-For
// and X-Real-IP headers for extracting the client IP address.
//
// WARNING: Only enable this if your gRPC server is deployed behind a trusted
// reverse proxy or load balancer that sanitizes these headers. Otherwise,
// malicious clients can easily spoof their IP address to bypass rate limits.
// By default, this is disabled for security.
func WithTrustProxy(trust bool) Option {
	return func(c *config) {
		c.trustProxy = trust
	}
}
