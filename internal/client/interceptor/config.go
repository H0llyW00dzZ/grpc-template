// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"google.golang.org/grpc/codes"
)

type tokenKey struct{}

// TokenSource enriches the context with a bearer token.
// It uses the same signature as server interceptor.AuthFunc for consistency.
type TokenSource func(ctx context.Context) (context.Context, error)

// StaticToken returns a TokenSource that adds a static bearer token to the context.
// Use this for development, testing, or services with long-lived credentials.
//
//	interceptor.Configure(
//	    interceptor.WithTokenSource(interceptor.StaticToken("my-api-key")),
//	)
func StaticToken(token string) TokenSource {
	return func(ctx context.Context) (context.Context, error) {
		return context.WithValue(ctx, tokenKey{}, token), nil
	}
}

// config holds shared configuration for all client interceptors in this package.
type config struct {
	logger         logging.Handler
	defaultTimeout time.Duration
	retryMax       int
	retryBackoff   time.Duration
	retryCodes     []codes.Code
	tokenSource    TokenSource
}

// defaultRetryCodes are the gRPC status codes that trigger a retry by default.
var defaultRetryCodes = []codes.Code{
	codes.Unavailable,
	codes.ResourceExhausted,
	codes.Aborted,
}

// defaultConfig is the package-level configuration used by all client interceptors.
var defaultConfig = &config{
	retryCodes: defaultRetryCodes,
}

// Option configures the client interceptor package.
type Option func(*config)

// WithLogger sets the logger used by client interceptors that perform logging
// (e.g., [Logging], [Retry]).
//
// If not set, interceptors fall back to [logging.Default].
func WithLogger(l logging.Handler) Option {
	return func(c *config) {
		c.logger = l
	}
}

// WithDefaultTimeout sets the default deadline applied by the [Timeout]
// interceptor when no deadline is already set on the context.
//
//	interceptor.Configure(
//	    interceptor.WithDefaultTimeout(5 * time.Second),
//	)
func WithDefaultTimeout(d time.Duration) Option {
	return func(c *config) {
		c.defaultTimeout = d
	}
}

// WithRetry configures the [Retry] interceptor's maximum attempts and
// base backoff duration. The actual backoff uses exponential growth with
// jitter: each attempt waits between backoff/2 and backoff * 2^attempt.
//
//	interceptor.Configure(
//	    interceptor.WithRetry(3, time.Second), // up to 3 retries, 1s base backoff
//	)
func WithRetry(maxRetries int, backoff time.Duration) Option {
	return func(c *config) {
		c.retryMax = maxRetries
		c.retryBackoff = backoff
	}
}

// WithRetryCodes overrides the default set of retryable gRPC status codes.
// By default, [codes.Unavailable], [codes.ResourceExhausted], and
// [codes.Aborted] are retried.
func WithRetryCodes(codes ...codes.Code) Option {
	return func(c *config) {
		c.retryCodes = codes
	}
}

// WithTokenSource sets the [TokenSource] used by [Auth] and [StreamAuth]
// to inject bearer tokens into outgoing metadata. The function may also
// enrich the context with claims or other metadata.
//
//	interceptor.Configure(
//	    interceptor.WithTokenSource(interceptor.StaticToken("my-token")),
//	)
func WithTokenSource(fn TokenSource) Option {
	return func(c *config) {
		c.tokenSource = fn
	}
}

// Configure applies the given options to the package-level client interceptor
// configuration. Call this once during application startup to share
// settings across all client interceptors.
//
//	interceptor.Configure(
//	    interceptor.WithLogger(myLogger),
//	    interceptor.WithDefaultTimeout(5 * time.Second),
//	    interceptor.WithRetry(3, time.Second),
//	)
//
// When using the [github.com/H0llyW00dzZ/grpc-template/internal/client] package,
// [github.com/H0llyW00dzZ/grpc-template/internal/client.WithLogger] calls this
// automatically—no manual configuration is needed.
func Configure(opts ...Option) {
	for _, opt := range opts {
		opt(defaultConfig)
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
