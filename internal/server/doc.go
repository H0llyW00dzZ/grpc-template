// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package server provides a high-level gRPC server with lifecycle management,
// functional options, and built-in health checking.
//
// # Creating a Server
//
// Use [New] with functional [Option] values to configure the server:
//
//	srv := server.New(
//	    server.WithPort("50051"),
//	    server.WithReflection(),
//	    server.WithLogger(myLogger),
//	    server.WithAuthFunc(myAuthFunc),
//	    server.WithExcludedMethods("/grpc.health.v1.Health/Check"),
//	    server.WithRateLimit(100, 200),
//	    server.WithTrustProxy(true), // only behind a trusted reverse proxy
//	    server.WithUnaryInterceptors(
//	        interceptor.Recovery(),
//	        interceptor.Logging(),
//	        interceptor.Auth(),
//	        interceptor.RateLimit(),
//	    ),
//	    server.WithStreamInterceptors(
//	        interceptor.StreamRecovery(),
//	        interceptor.StreamLogging(),
//	        interceptor.StreamAuth(),
//	        interceptor.StreamRateLimit(),
//	    ),
//	)
//
// Options that accept shared dependencies (logger, auth function, excluded
// methods, rate limit) automatically delegate to [interceptor.Configure],
// keeping configuration in a single place.
//
// # Registering Services
//
// Use [Server.RegisterService] to attach gRPC service implementations:
//
//	greeterSvc := greeter.NewService(srv.Logger())
//	srv.RegisterService(greeterSvc.Register)
//
// # Running and Shutdown
//
// [Server.Run] starts the server and blocks until the context is cancelled
// or a SIGINT/SIGTERM signal is received. The server performs a graceful
// shutdown, draining in-flight RPCs before stopping:
//
//	if err := srv.Run(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
// # Health Checking
//
// The server automatically registers the standard [gRPC Health Checking Protocol].
// Use [Server.Health] to toggle per-service health status at runtime:
//
//	// Take a service offline for maintenance:
//	srv.Health().SetServingStatus(
//	    "helloworld.v1.GreeterService",
//	    healthgrpc.HealthCheckResponse_NOT_SERVING,
//	)
//
// On graceful shutdown, the overall health status is automatically set
// to NOT_SERVING before draining connections.
//
// # Available Options
//
//   - [WithPort] — TCP port to listen on (default "50051")
//   - [WithReflection] — enable gRPC server reflection
//   - [WithTLS] / [WithMutualTLS] — TLS and mutual TLS
//   - [WithLogger] — pluggable logger (syncs to interceptors)
//   - [WithAuthFunc] — authentication function (syncs to interceptors)
//   - [WithExcludedMethods] — methods to skip auth (syncs to interceptors)
//   - [WithUnaryInterceptors] / [WithStreamInterceptors] — interceptor chains
//   - [WithRateLimit] — per-peer rate limiting (syncs to interceptors)
//   - [WithTrustProxy] — trust X-Forwarded-For / X-Real-IP behind proxies (syncs to interceptors)
//   - [WithKeepalive] — connection keepalive parameters
//   - [WithMaxMsgSize] — maximum message size
//   - [WithMaxConcurrentStreams] — concurrent stream limit
//   - [WithGrpcOptions] — raw grpc.ServerOption pass-through
//   - [WithListener] — custom net.Listener (e.g., bufconn for testing)
//
// [gRPC Health Checking Protocol]: https://github.com/grpc/grpc/blob/master/doc/health-checking.md
package server
