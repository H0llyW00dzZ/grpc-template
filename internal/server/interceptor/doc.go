// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package interceptor provides modular gRPC server interceptors for both
// unary and streaming RPCs.
//
// Each interceptor is a standalone function that returns a
// [grpc.UnaryServerInterceptor] or [grpc.StreamServerInterceptor], and
// can be composed via [server.WithUnaryInterceptors] and
// [server.WithStreamInterceptors].
//
// # Shared Configuration
//
// Interceptors read shared dependencies (logger, auth function, excluded
// methods, rate limit) from a package-level configuration. Use [Configure]
// with functional options to set up the shared state once at startup:
//
//	interceptor.Configure(
//	    interceptor.WithLogger(myLogger),
//	    interceptor.WithAuthFunc(myAuthFunc),
//	    interceptor.WithExcludedMethods("/grpc.health.v1.Health/Check"),
//	    interceptor.WithDemotedMethods("/myapp.v1.LongPoll/Watch"), // extend built-in defaults
//	    interceptor.WithRateLimit(100, 200), // uses default in-memory rate limiter
//	    // interceptor.WithRateLimiter(myRedisLimiter), // or inject a custom backend!
//	    interceptor.WithTrustProxy(true), // only behind a trusted reverse proxy
//	)
//
// When using the [server] package, the equivalent server options
// ([server.WithLogger], [server.WithAuthFunc], [server.WithExcludedMethods],
// [server.WithDemotedMethods], [server.WithRateLimit], [server.WithTrustProxy])
// call [Configure] automatically — no manual setup needed.
//
// # Thread Safety
//
// All interceptors read their configuration through a snapshot taken under
// a read lock, so [Configure] may be called concurrently with in-flight
// RPCs without data races. Each interceptor uses a single config snapshot
// for the entire request, including derived operations like peer key
// extraction, ensuring consistency within a request. Interceptors should
// resolve the logger using [logging.Resolve].
//
// # Available Interceptors
//
//   - [Logging] / [StreamLogging] — logs method, duration, gRPC status code,
//     peer address, request ID, and error for every RPC. When [WithTrustProxy]
//     is enabled, the logged peer reflects the true client IP from proxy headers.
//     Methods configured via [WithDemotedMethods] have their [codes.Canceled]
//     errors demoted from Error to Debug level. gRPC reflection methods are
//     demoted by default; the option is additive (extends, never replaces).
//   - [Recovery] / [StreamRecovery] — recovers from panics and returns
//     codes.Internal to the client.
//   - [RequestID] / [StreamRequestID] — extracts or generates a unique
//     request ID (x-request-id) for distributed tracing. Incoming values
//     are validated against a strict UUID format (8-4-4-4-12 hex);
//     non-matching or missing values are replaced with a server-generated
//     UUID to prevent spoofing and log injection. Retrieve the ID
//     downstream with [RequestIDFromContext].
//   - [Auth] / [StreamAuth] — validates bearer tokens via a pluggable
//     [AuthFunc] with support for method exclusion.
//   - [Validation] — validates incoming requests implementing the
//     [Validator] interface (compatible with protoc-gen-validate / buf validate).
//   - [RateLimit] / [StreamRateLimit] — configurable per-peer rate limiting
//     powered by a [RateLimiter] interface (scalable to Redis or other databases).
//     The default [MemoryRateLimiter] (created via [NewMemoryRateLimiter])
//     executes a token-bucket algorithm with automatic stale-limiter cleanup
//     (at half the TTL interval for tighter memory reclamation). When a rate limiter is replaced via
//     [WithRateLimiter] or [WithRateLimit], the previous limiter is stopped
//     automatically if it implements a Stop method (e.g., [MemoryRateLimiter]),
//     preventing background goroutine leaks. Supports proxy-aware client IP
//     extraction via [WithTrustProxy] (X-Forwarded-For, X-Real-IP).
//
// # Benchmarks
//
// The package includes benchmarks for every interceptor and the config
// snapshot mechanism. Run them with:
//
//	go test ./internal/server/interceptor -bench=. -benchmem
//
// Or filter to a specific interceptor:
//
//	go test ./internal/server/interceptor -bench=BenchmarkRecovery -benchmem
//
// # Usage
//
//	srv := server.New(
//		server.WithLogger(myLogger),
//		server.WithAuthFunc(myAuthFunc),
//		server.WithExcludedMethods("/grpc.health.v1.Health/Check"),
//		server.WithRateLimit(100, 200),
//		server.WithUnaryInterceptors(
//			interceptor.Recovery(),
//			interceptor.Logging(),
//			interceptor.RequestID(),
//			interceptor.Auth(),
//			interceptor.Validation(),
//			interceptor.RateLimit(),
//		),
//		server.WithStreamInterceptors(
//			interceptor.StreamRecovery(),
//			interceptor.StreamLogging(),
//			interceptor.StreamRequestID(),
//			interceptor.StreamAuth(),
//			interceptor.StreamRateLimit(),
//		),
//	)
package interceptor
