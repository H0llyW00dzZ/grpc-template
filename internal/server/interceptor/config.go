// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import "github.com/H0llyW00dzZ/grpc-template/internal/logging"

// config holds shared configuration for all interceptors in this package.
type config struct {
	logger logging.Handler
}

// defaultConfig is the package-level configuration used by all interceptors.
var defaultConfig = &config{}

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

// Configure applies the given options to the package-level interceptor
// configuration. Call this once during application startup to share
// settings (such as a logger) across all interceptors.
//
//	interceptor.Configure(
//	    interceptor.WithLogger(myLogger),
//	)
//
// When using the [server] package, [server.WithLogger] calls this
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
