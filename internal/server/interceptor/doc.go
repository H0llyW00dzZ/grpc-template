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
// Interceptors that need shared dependencies (such as a logger) read them
// from a package-level configuration. Use [Configure] with functional options
// to set up the shared state once at startup:
//
//	interceptor.Configure(
//	    interceptor.WithLogger(myLogger),
//	)
//
// When using the [server] package, [server.WithLogger] calls [Configure]
// automatically — no manual setup needed.
//
// # Available Interceptors
//
//   - [Logging] / [StreamLogging] — logs method, duration, gRPC status code,
//     peer address, request ID, and error for every RPC.
//   - [Recovery] / [StreamRecovery] — recovers from panics and returns
//     codes.Internal to the client.
//   - [RequestID] / [StreamRequestID] — extracts or generates a unique
//     request ID (x-request-id) for distributed tracing.
//   - [Auth] / [StreamAuth] — validates bearer tokens via a pluggable
//     [AuthFunc] with support for method exclusion.
//   - [Validation] — validates incoming requests implementing the
//     [Validator] interface (compatible with protoc-gen-validate / buf validate).
//
// # Usage
//
//	srv := server.New(
//		server.WithLogger(myLogger), // syncs to interceptor package automatically
//		server.WithUnaryInterceptors(
//			interceptor.Recovery(),
//			interceptor.Logging(),
//			interceptor.RequestID(),
//			interceptor.Auth(myAuthFunc, interceptor.WithExcludedMethods("/grpc.health.v1.Health/Check")),
//			interceptor.Validation(),
//		),
//		server.WithStreamInterceptors(
//			interceptor.StreamRecovery(),
//			interceptor.StreamLogging(),
//			interceptor.StreamRequestID(),
//			interceptor.StreamAuth(myAuthFunc),
//		),
//	)
package interceptor
