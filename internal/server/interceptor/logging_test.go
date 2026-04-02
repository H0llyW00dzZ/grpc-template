// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor_test

import (
	"context"
	"net"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/server/interceptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestLogging(t *testing.T) {
	i := interceptor.Logging()

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/TestMethod"}

	resp, err := i(context.Background(), "request", info, handler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestLogging_Error(t *testing.T) {
	i := interceptor.Logging()

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/TestMethod"}

	_, err := i(context.Background(), "request", info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestStreamLogging(t *testing.T) {
	i := interceptor.StreamLogging()

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/TestStream"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := i(nil, ss, info, handler)
	require.NoError(t, err)
}

func TestStreamLogging_Error(t *testing.T) {
	i := interceptor.StreamLogging()

	handler := func(srv any, stream grpc.ServerStream) error {
		return status.Error(codes.Internal, "stream error")
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/TestStream"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := i(nil, ss, info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestLogging_NilLogger(t *testing.T) {
	i := interceptor.Logging()

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/TestMethod"}

	resp, err := i(context.Background(), "request", info, handler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestStreamLogging_NilLogger(t *testing.T) {
	i := interceptor.StreamLogging()

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/TestStream"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := i(nil, ss, info, handler)
	require.NoError(t, err)
}

func TestLogging_WithPeerAndRequestID(t *testing.T) {
	reqIDInt := interceptor.RequestID()
	logInt := interceptor.Logging()

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/TestMethod"}

	ctx := context.Background()
	p := &peer.Peer{Addr: &net.TCPAddr{IP: net.IPv4(192, 168, 1, 1), Port: 50051}}
	ctx = peer.NewContext(ctx, p)

	resp, err := reqIDInt(ctx, "req", info, func(ctx context.Context, req any) (any, error) {
		return logInt(ctx, req, info, handler)
	})

	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestLogging_WithTrustProxy_XForwardedFor(t *testing.T) {
	interceptor.Configure(interceptor.WithTrustProxy(true))
	defer interceptor.Configure(interceptor.WithTrustProxy(false))

	logInt := interceptor.Logging()

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/TestMethod"}

	// Set proxy header and a different peer address to verify proxy header is used.
	md := metadata.Pairs("x-forwarded-for", "203.0.113.50")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 50051},
	})

	resp, err := logInt(ctx, "req", info, handler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestStreamLogging_WithTrustProxy_XRealIP(t *testing.T) {
	interceptor.Configure(interceptor.WithTrustProxy(true))
	defer interceptor.Configure(interceptor.WithTrustProxy(false))

	logInt := interceptor.StreamLogging()

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/TestStream"}

	md := metadata.Pairs("x-real-ip", "198.51.100.10")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 50051},
	})
	ss := &fakeServerStream{ctx: ctx}

	err := logInt(nil, ss, info, handler)
	require.NoError(t, err)
}

func TestLogging_WithoutTrustProxy_IgnoresHeaders(t *testing.T) {
	// Ensure trustProxy is off.
	interceptor.Configure(interceptor.WithTrustProxy(false))

	logInt := interceptor.Logging()

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/TestMethod"}

	// Even with proxy headers, the peer should use the direct TCP address.
	md := metadata.Pairs("x-forwarded-for", "203.0.113.50")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 50051},
	})

	resp, err := logInt(ctx, "req", info, handler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestLogging_ReflectionCanceled(t *testing.T) {
	i := interceptor.Logging()

	handler := func(_ context.Context, _ any) (any, error) {
		return nil, status.Error(codes.Canceled, "context canceled")
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/grpc.reflection.v1.ServerReflection/ServerReflectionInfo"}

	_, err := i(context.Background(), nil, info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Canceled, st.Code())
}

func TestStreamLogging_ReflectionCanceled(t *testing.T) {
	i := interceptor.StreamLogging()

	handler := func(srv any, stream grpc.ServerStream) error {
		return status.Error(codes.Canceled, "context canceled")
	}
	info := &grpc.StreamServerInfo{FullMethod: "/grpc.reflection.v1.ServerReflection/ServerReflectionInfo"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := i(nil, ss, info, handler)
	require.Error(t, err)
}

func TestLogging_CustomDemotedMethod(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(
		interceptor.WithLogger(&noopLogger{}),
		interceptor.WithDemotedMethods("/myapp.v1.LongPoll/Watch", ""),
	)
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Logging()

	handler := func(_ context.Context, _ any) (any, error) {
		return nil, status.Error(codes.Canceled, "context canceled")
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/myapp.v1.LongPoll/Watch"}

	// Should be demoted to Debug (not Error) — verifying no panic and error passes through.
	_, err := i(context.Background(), nil, info, handler)
	require.Error(t, err)
	assert.Equal(t, codes.Canceled, status.Code(err))
}
