// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor_test

import (
	"context"
	"testing"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/client/interceptor"
	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// BenchmarkClientGetConfig measures the cost of taking a config snapshot
// under the read lock on the client interceptor package.
func BenchmarkClientGetConfig(b *testing.B) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithLogger(logging.Default()))
	b.Cleanup(interceptor.ResetConfig)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = interceptor.TestableLogger()
	}
}

// BenchmarkClientLoggingInterceptor measures the full unary client logging
// interceptor including config snapshot, invoker call, and log emission.
func BenchmarkClientLoggingInterceptor(b *testing.B) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithLogger(&noopLogger{}))
	b.Cleanup(interceptor.ResetConfig)

	i := interceptor.Logging()
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return nil
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = i(context.Background(), "/bench.v1.Service/Method", nil, nil, nil, invoker)
	}
}

// BenchmarkClientAuthInterceptor_NoToken measures the auth interceptor
// overhead when no token source is configured (no-op fast path).
func BenchmarkClientAuthInterceptor_NoToken(b *testing.B) {
	interceptor.ResetConfig()
	b.Cleanup(interceptor.ResetConfig)

	i := interceptor.Auth()
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return nil
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = i(context.Background(), "/bench.v1.Service/Method", nil, nil, nil, invoker)
	}
}

// BenchmarkClientAuthInterceptor_WithToken measures the auth interceptor
// with a static token source injecting a bearer token.
func BenchmarkClientAuthInterceptor_WithToken(b *testing.B) {
	interceptor.ResetConfig()
	interceptor.Configure(
		interceptor.WithLogger(&noopLogger{}),
		interceptor.WithTokenSource(interceptor.StaticToken("bench-token")),
	)
	b.Cleanup(interceptor.ResetConfig)

	i := interceptor.Auth()
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return nil
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = i(context.Background(), "/bench.v1.Service/Method", nil, nil, nil, invoker)
	}
}

// BenchmarkClientAuthInterceptor_ExistingMetadata measures token injection
// when the context already carries outgoing metadata.
func BenchmarkClientAuthInterceptor_ExistingMetadata(b *testing.B) {
	interceptor.ResetConfig()
	interceptor.Configure(
		interceptor.WithLogger(&noopLogger{}),
		interceptor.WithTokenSource(interceptor.StaticToken("bench-token")),
	)
	b.Cleanup(interceptor.ResetConfig)

	i := interceptor.Auth()
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return nil
	}
	md := metadata.Pairs("x-custom", "value")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = i(ctx, "/bench.v1.Service/Method", nil, nil, nil, invoker)
	}
}

// BenchmarkClientRetryInterceptor_NoRetry measures the retry interceptor
// overhead when retries are disabled (no-op fast path).
func BenchmarkClientRetryInterceptor_NoRetry(b *testing.B) {
	interceptor.ResetConfig()
	b.Cleanup(interceptor.ResetConfig)

	i := interceptor.Retry()
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return nil
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = i(context.Background(), "/bench.v1.Service/Method", nil, nil, nil, invoker)
	}
}

// BenchmarkClientTimeoutInterceptor measures the timeout interceptor
// overhead when applying a default deadline.
func BenchmarkClientTimeoutInterceptor(b *testing.B) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithDefaultTimeout(5 * time.Second))
	b.Cleanup(interceptor.ResetConfig)

	i := interceptor.Timeout()
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return nil
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = i(context.Background(), "/bench.v1.Service/Method", nil, nil, nil, invoker)
	}
}

// BenchmarkBackoffDuration measures the exponential backoff calculation.
func BenchmarkBackoffDuration(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = interceptor.BackoffDuration(3, time.Second)
	}
}

// noopLogger is a zero-cost logger for benchmarks that discards all output.
type noopLogger struct{}

func (*noopLogger) Debug(_ string, _ ...any) {}
func (*noopLogger) Info(_ string, _ ...any)  {}
func (*noopLogger) Warn(_ string, _ ...any)  {}
func (*noopLogger) Error(_ string, _ ...any) {}
