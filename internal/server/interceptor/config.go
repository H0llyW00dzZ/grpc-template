// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"golang.org/x/time/rate"
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
	rateLimit       rate.Limit
	rateBurst       int
	rateLimitTTL    time.Duration
}

// defaultConfig is the package-level configuration used by all interceptors.
var defaultConfig = &config{
	excludedMethods: make(map[string]struct{}),
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
	for _, opt := range opts {
		opt(defaultConfig)
	}
}

// WithRateLimit sets the per-peer rate limit in requests per second and
// the burst size (maximum number of requests allowed at once).
// A rate of 0 or negative disables rate limiting.
//
//	interceptor.Configure(
//	    interceptor.WithRateLimit(100, 200), // 100 req/s, burst up to 200
//	)
func WithRateLimit(rps float64, burst int) Option {
	return func(c *config) {
		c.rateLimit = rate.Limit(rps)
		c.rateBurst = burst
	}
}

// WithRateLimitTTL sets the duration after which idle per-peer limiters
// are removed from memory. Default is 10 minutes.
func WithRateLimitTTL(ttl time.Duration) Option {
	return func(c *config) {
		c.rateLimitTTL = ttl
	}
}

// logger returns the configured logger, falling back to [logging.Default]
// if none has been set via [Configure].
func logger() logging.Handler {
	if defaultConfig.logger != nil {
		return defaultConfig.logger
	}
	return logging.Default()
}
