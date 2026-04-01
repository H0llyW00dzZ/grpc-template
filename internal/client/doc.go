// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package client provides a high-level gRPC client with lifecycle management,
// functional options, and built-in health watching.
//
// It is the client-side counterpart of
// [github.com/H0llyW00dzZ/grpc-template/internal/server], using the same
// functional-options architecture.
//
// # Creating a Client
//
// Use [New] with functional [Option] values to configure the client,
// then call [Client.Connect] to establish the connection:
//
//	c := client.New("localhost:50051",
//	    client.WithInsecure(),
//	    client.WithLogger(myLogger),
//	    client.WithDefaultTimeout(5 * time.Second),
//	    client.WithRetry(3, time.Second),
//	    client.WithUnaryInterceptors(
//	        interceptor.Logging(),
//	        interceptor.Timeout(),
//	        interceptor.Retry(),
//	        interceptor.Auth(),
//	    ),
//	    client.WithStreamInterceptors(
//	        interceptor.StreamLogging(),
//	        interceptor.StreamAuth(),
//	    ),
//	)
//
//	if err := c.Connect(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer c.Close()
//
// Options that accept shared dependencies (logger, timeout, retry,
// token source) automatically delegate to
// [github.com/H0llyW00dzZ/grpc-template/internal/client/interceptor.Configure],
// keeping configuration in a single place.
//
// # Using Service Stubs
//
// After connecting, use [Client.Conn] to obtain the underlying
// [grpc.ClientConn] and create service stubs:
//
//	greeter := pb.NewGreeterServiceClient(c.Conn())
//	reply, err := greeter.SayHello(ctx, &pb.SayHelloRequest{Name: "World"})
//
// Or use the higher-level caller wrappers in [internal/service]:
//
//	caller := greeter.NewCaller(c.Conn(), c.Logger())
//	reply, err := caller.SayHello(ctx, "World")
//
// # Connection Lifecycle
//
// [Client.Connect] creates the gRPC connection using [grpc.NewClient],
// which connects lazily. It also validates any deferred configuration
// errors (e.g., invalid TLS certificates) before dialling. Calling
// Connect on an already-connected client returns an error; call
// [Client.Close] first to reconnect.
// To block until the connection is ready, use [Client.WaitReady]:
//
//	if err := c.Connect(ctx); err != nil { ... }
//	if err := c.WaitReady(ctx); err != nil { ... }
//
// If neither [WithTLS], [WithMutualTLS], nor [WithInsecure] is set,
// the client defaults to insecure credentials and logs a warning.
//
// [Client.Close] gracefully shuts down the connection and cancels
// any background goroutines (e.g., health watching).
//
// # Health Watching
//
// Enable [WithHealthWatch] to monitor the server's health status
// in a background goroutine after connecting. The watcher automatically
// reconnects with exponential backoff (500ms to 30s) if the health
// stream is interrupted:
//
//	c := client.New("localhost:50051",
//	    client.WithInsecure(),
//	    client.WithHealthWatch(),
//	)
//
// # Available Options
//
//   - [WithInsecure] — disable transport security (dev/testing); clears any prior TLS config error
//   - [WithTLS] — TLS with server CA verification (errors deferred to [Client.Connect])
//   - [WithMutualTLS] — mutual TLS for service-to-service (errors deferred to [Client.Connect])
//   - [WithLogger] — pluggable logger (syncs to interceptors)
//   - [WithUnaryInterceptors] / [WithStreamInterceptors] — interceptor chains
//   - [WithDefaultTimeout] — default RPC deadline (syncs to interceptors)
//   - [WithRetry] — retry on transient failures (syncs to interceptors)
//   - [WithRetryCodes] — override retryable status codes (syncs to interceptors)
//   - [WithTokenSource] — bearer token injection (syncs to interceptors, supports [StaticToken] and [OAuth2TokenSource])
//   - [WithHealthWatch] — background health monitoring with auto-reconnect
//   - [WithKeepalive] — connection keepalive parameters
//   - [WithMaxMsgSize] — maximum message size
//   - [WithDialOptions] — raw grpc.DialOption pass-through
package client
