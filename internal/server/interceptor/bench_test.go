// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor_test

import (
	"context"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/server/interceptor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// BenchmarkGetConfig measures the cost of taking a config snapshot under
// the read lock via a minimal interceptor call. This is the hot path
// executed on every RPC.
func BenchmarkGetConfig(b *testing.B) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithLogger(&noopLogger{}))
	b.Cleanup(interceptor.ResetConfig)

	// Recovery() is the cheapest interceptor — just getConfig() + deferred recover.
	i := interceptor.Recovery()
	handler := func(_ context.Context, _ any) (any, error) {
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/bench.v1.Service/GetConfig"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = i(context.Background(), nil, info, handler)
	}
}

// BenchmarkLoggingInterceptor measures the full unary logging interceptor
// including config snapshot, handler invocation, and structured log emission.
func BenchmarkLoggingInterceptor(b *testing.B) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithLogger(&noopLogger{}))
	b.Cleanup(interceptor.ResetConfig)

	i := interceptor.Logging()
	handler := func(_ context.Context, _ any) (any, error) {
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/bench.v1.Service/Method"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = i(context.Background(), nil, info, handler)
	}
}

// BenchmarkRecoveryInterceptor measures the recovery interceptor overhead
// when no panic occurs (the typical path).
func BenchmarkRecoveryInterceptor(b *testing.B) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithLogger(&noopLogger{}))
	b.Cleanup(interceptor.ResetConfig)

	i := interceptor.Recovery()
	handler := func(_ context.Context, _ any) (any, error) {
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/bench.v1.Service/Method"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = i(context.Background(), nil, info, handler)
	}
}

// BenchmarkAuthInterceptor_NoAuth measures the auth interceptor overhead
// when no auth function is configured (no-op fast path).
func BenchmarkAuthInterceptor_NoAuth(b *testing.B) {
	interceptor.ResetConfig()
	b.Cleanup(interceptor.ResetConfig)

	i := interceptor.Auth()
	handler := func(_ context.Context, _ any) (any, error) {
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/bench.v1.Service/Method"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = i(context.Background(), nil, info, handler)
	}
}

// BenchmarkAuthInterceptor_WithAuth measures the auth interceptor with a
// real auth function and valid bearer token.
func BenchmarkAuthInterceptor_WithAuth(b *testing.B) {
	interceptor.ResetConfig()
	interceptor.Configure(
		interceptor.WithLogger(&noopLogger{}),
		interceptor.WithAuthFunc(func(ctx context.Context, token string) (context.Context, error) {
			return ctx, nil
		}),
	)
	b.Cleanup(interceptor.ResetConfig)

	i := interceptor.Auth()
	handler := func(_ context.Context, _ any) (any, error) {
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/bench.v1.Service/Method"}
	md := metadata.Pairs("authorization", "Bearer bench-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = i(ctx, nil, info, handler)
	}
}

// BenchmarkRateLimitInterceptor measures per-peer rate limiting on the hot
// path (single peer, always allowed).
func BenchmarkRateLimitInterceptor(b *testing.B) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithRateLimit(1e9, 1e9)) // effectively unlimited
	b.Cleanup(func() {
		interceptor.StopCleanup()
		interceptor.ResetConfig()
	})

	i := interceptor.RateLimit()
	handler := func(_ context.Context, _ any) (any, error) {
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/bench.v1.Service/Method"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = i(context.Background(), nil, info, handler)
	}
}

// BenchmarkRateLimitInterceptor_Disabled measures the no-op path when
// no rate limiter is configured.
func BenchmarkRateLimitInterceptor_Disabled(b *testing.B) {
	interceptor.ResetConfig()
	b.Cleanup(interceptor.ResetConfig)

	i := interceptor.RateLimit()
	handler := func(_ context.Context, _ any) (any, error) {
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/bench.v1.Service/Method"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = i(context.Background(), nil, info, handler)
	}
}

// BenchmarkPeerKey measures client IP extraction from gRPC peer info.
func BenchmarkPeerKey(b *testing.B) {
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = interceptor.PeerKey(ctx, false)
	}
}

// BenchmarkPeerKey_TrustProxy measures client IP extraction with proxy
// header inspection enabled.
func BenchmarkPeerKey_TrustProxy(b *testing.B) {
	md := metadata.Pairs("x-forwarded-for", "10.0.0.1, 192.168.1.1")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = interceptor.PeerKey(ctx, true)
	}
}

// noopLogger is a zero-cost logger for benchmarks that discards all output.
type noopLogger struct{}

func (*noopLogger) Debug(_ string, _ ...any) {}
func (*noopLogger) Info(_ string, _ ...any)  {}
func (*noopLogger) Warn(_ string, _ ...any)  {}
func (*noopLogger) Error(_ string, _ ...any) {}
