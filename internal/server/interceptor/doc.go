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
//	    interceptor.WithRateLimit(100, 200),
//	    interceptor.WithTrustProxy(true), // only behind a trusted reverse proxy
//	)
//
// When using the [server] package, the equivalent server options
// ([server.WithLogger], [server.WithAuthFunc], [server.WithExcludedMethods],
// [server.WithRateLimit], [server.WithTrustProxy]) call [Configure]
// automatically — no manual setup needed.
//
// # Available Interceptors
//
//   - [Logging] / [StreamLogging] — logs method, duration, gRPC status code,
//     peer address, request ID, and error for every RPC. When [WithTrustProxy]
//     is enabled, the logged peer reflects the true client IP from proxy headers.
//   - [Recovery] / [StreamRecovery] — recovers from panics and returns
//     codes.Internal to the client.
//   - [RequestID] / [StreamRequestID] — extracts or generates a unique
//     request ID (x-request-id) for distributed tracing.
//   - [Auth] / [StreamAuth] — validates bearer tokens via a pluggable
//     [AuthFunc] with support for method exclusion.
//   - [Validation] — validates incoming requests implementing the
//     [Validator] interface (compatible with protoc-gen-validate / buf validate).
//   - [RateLimit] / [StreamRateLimit] — per-peer token-bucket rate limiting
//     with automatic stale-limiter cleanup. Supports proxy-aware client IP
//     extraction via [WithTrustProxy] (X-Forwarded-For, X-Real-IP).
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
