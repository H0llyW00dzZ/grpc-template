// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package interceptor provides client-side gRPC interceptors
// for logging, timeouts, retries, and authentication.
//
// The package mirrors the server-side [github.com/H0llyW00dzZ/grpc-template/internal/server/interceptor]
// architecture, using a shared package-level configuration set via [Configure].
//
// # Configuration
//
// Call [Configure] once during application startup to share settings
// (such as a logger) across all client interceptors:
//
//	interceptor.Configure(
//	    interceptor.WithLogger(myLogger),
//	    interceptor.WithDefaultTimeout(5 * time.Second),
//	    interceptor.WithRetry(3, time.Second),
//	    interceptor.WithRetryCodes(codes.Unavailable, codes.ResourceExhausted),
//	    interceptor.WithTokenSource(interceptor.StaticToken("my-token")),
//	    // interceptor.WithTokenSource(interceptor.OAuth2TokenSource(oauth2Src)),
//	)
//
// When using the [github.com/H0llyW00dzZ/grpc-template/internal/client] package,
// options like [github.com/H0llyW00dzZ/grpc-template/internal/client.WithLogger]
// call [Configure] automatically—no manual configuration is needed.
//
// # Thread Safety
//
// All interceptors read their configuration through a snapshot taken under
// a read lock, so [Configure] may be called concurrently with in-flight
// RPCs without data races. Interceptors resolve the logger via the
// snapshot's resolvedLogger method, which falls back to [logging.Default]
// when no logger has been configured.
//
// # Available Interceptors
//
//   - [Logging] / [StreamLogging] — logs RPC method, duration, and status
//   - [Timeout] — enforces a default deadline on unary RPCs
//   - [Retry] — retries transient failures with exponential backoff and jitter
//   - [Auth] / [StreamAuth] — injects bearer tokens into outgoing metadata
//     (use [OAuth2TokenSource] for golang.org/x/oauth2 support)
package interceptor
