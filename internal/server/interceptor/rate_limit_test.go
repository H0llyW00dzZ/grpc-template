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
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestRateLimit_Allowed(t *testing.T) {
	interceptor.Configure(interceptor.WithRateLimit(100, 10))

	i := interceptor.RateLimit()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Method"}

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	ctx := peerContext("192.168.1.1:12345")

	resp, err := i(ctx, "req", info, handler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestRateLimit_Exceeded(t *testing.T) {
	// Allow only 1 request per second with burst of 1.
	interceptor.Configure(interceptor.WithRateLimit(1, 1))

	i := interceptor.RateLimit()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Method"}

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	// Use a unique IP to avoid limiter reuse from other tests.
	ctx := peerContext("10.0.0.1:12345")

	// First request should pass (consumes the 1-burst token).
	_, err := i(ctx, "req", info, handler)
	require.NoError(t, err)

	// Second request should be rate limited.
	_, err = i(ctx, "req", info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st.Code())
	assert.Equal(t, "rate limit exceeded", st.Message())
}

func TestRateLimit_PerPeer(t *testing.T) {
	// Allow 1 request per second with burst of 1.
	interceptor.Configure(interceptor.WithRateLimit(1, 1))

	i := interceptor.RateLimit()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Method"}

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	ctx1 := peerContext("10.1.1.1:12345")
	ctx2 := peerContext("10.2.2.2:12345")

	// Peer 1 uses its burst token.
	_, err := i(ctx1, "req", info, handler)
	require.NoError(t, err)

	// Peer 2 should still be allowed (independent limiter).
	_, err = i(ctx2, "req", info, handler)
	require.NoError(t, err)

	// Peer 1 is now rate-limited.
	_, err = i(ctx1, "req", info, handler)
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.ResourceExhausted, st.Code())
}

func TestRateLimit_NoPeer(t *testing.T) {
	interceptor.Configure(interceptor.WithRateLimit(1, 1))

	i := interceptor.RateLimit()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Method"}

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	// Context without peer info — should fall back to "unknown" key.
	resp, err := i(context.Background(), "req", info, handler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestRateLimit_Disabled(t *testing.T) {
	// Disable rate limiting by setting rate to 0.
	interceptor.Configure(interceptor.WithRateLimit(0, 0))

	i := interceptor.RateLimit()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Method"}

	handler := func(ctx context.Context, req any) (any, error) {
		return "passthrough", nil
	}

	ctx := peerContext("10.3.3.3:12345")

	// Should always pass through when disabled.
	for range 10 {
		resp, err := i(ctx, "req", info, handler)
		require.NoError(t, err)
		assert.Equal(t, "passthrough", resp)
	}
}

func TestStreamRateLimit_Allowed(t *testing.T) {
	interceptor.Configure(interceptor.WithRateLimit(100, 10))

	i := interceptor.StreamRateLimit()
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.Svc/StreamMethod"}

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}

	ctx := peerContext("10.4.4.4:12345")
	ss := &fakeServerStream{ctx: ctx}

	err := i(nil, ss, info, handler)
	require.NoError(t, err)
}

func TestStreamRateLimit_Exceeded(t *testing.T) {
	interceptor.Configure(interceptor.WithRateLimit(1, 1))

	i := interceptor.StreamRateLimit()
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.Svc/StreamMethod"}

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}

	ctx := peerContext("10.5.5.5:12345")
	ss := &fakeServerStream{ctx: ctx}

	// First request passes.
	err := i(nil, ss, info, handler)
	require.NoError(t, err)

	// Second request is rate limited.
	err = i(nil, ss, info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st.Code())
}

// peerContext creates a context with the given address as peer info.
func peerContext(addr string) context.Context {
	return peer.NewContext(context.Background(), &peer.Peer{
		Addr: fakeAddr(addr),
	})
}

// fakeAddr implements net.Addr for testing.
type fakeAddr string

func (a fakeAddr) Network() string { return "tcp" }
func (a fakeAddr) String() string  { return string(a) }

// Ensure fakeAddr implements net.Addr.
var _ net.Addr = fakeAddr("")
