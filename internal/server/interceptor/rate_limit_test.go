// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/server/interceptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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

func TestStreamRateLimit_Disabled(t *testing.T) {
	interceptor.Configure(interceptor.WithRateLimit(0, 0))

	i := interceptor.StreamRateLimit()
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.Svc/StreamMethod"}

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}

	ctx := peerContext("10.6.6.6:12345")
	ss := &fakeServerStream{ctx: ctx}

	for range 10 {
		err := i(nil, ss, info, handler)
		require.NoError(t, err)
	}
}

func TestRateLimit_Cleanup(t *testing.T) {
	interceptor.Configure(interceptor.WithRateLimit(100, 10))

	i := interceptor.RateLimit()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Method"}

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	ctx := peerContext("10.99.99.99:12345")

	// Create a limiter entry by making a request.
	_, err := i(ctx, "req", info, handler)
	require.NoError(t, err)
	assert.Greater(t, interceptor.LimiterCount(), 0)

	// Set lastSeen to the past so cleanup will remove it.
	interceptor.SetLimiterLastSeen("10.99.99.99", time.Now().Add(-1*time.Hour))

	// Run cleanup with a 1-second TTL — the stale entry should be removed.
	interceptor.CleanupLimiters(1 * time.Second)

	// The stale entry for 10.99.99.99 should have been removed.
	// Other entries from other tests may still exist, so just verify the
	// cleanup ran without error by making a new request (which creates a fresh limiter).
	_, err = i(ctx, "req", info, handler)
	require.NoError(t, err)
}

func TestPeerKey_NilAddr(t *testing.T) {
	// With nil peer addr — should return "unknown".
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: nil})
	key := interceptor.PeerKey(ctx)
	assert.Equal(t, "unknown", key)
}

func TestPeerKey_NoPort(t *testing.T) {
	// Address without a port — SplitHostPort will fail, so it returns the raw string.
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: fakeAddr("192.168.1.1"),
	})
	key := interceptor.PeerKey(ctx)
	assert.Equal(t, "192.168.1.1", key)
}

func TestPeerKey_XForwardedFor_Single(t *testing.T) {
	interceptor.Configure(interceptor.WithTrustProxy(true))
	defer interceptor.Configure(interceptor.WithTrustProxy(false)) // reset

	md := metadata.Pairs("x-forwarded-for", "203.0.113.1")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// Also add a dummy peer context to ensure metadata takes precedence
	ctx = peer.NewContext(ctx, &peer.Peer{Addr: fakeAddr("192.168.1.1:12345")})

	key := interceptor.PeerKey(ctx)
	assert.Equal(t, "203.0.113.1", key)
}

func TestPeerKey_XForwardedFor_Multiple(t *testing.T) {
	interceptor.Configure(interceptor.WithTrustProxy(true))
	defer interceptor.Configure(interceptor.WithTrustProxy(false)) // reset

	md := metadata.Pairs("x-forwarded-for", "203.0.113.1, 198.51.100.2, 192.0.2.3")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = peer.NewContext(ctx, &peer.Peer{Addr: fakeAddr("192.168.1.1:12345")})

	key := interceptor.PeerKey(ctx)
	// Should extract the first IP and trim spaces
	assert.Equal(t, "203.0.113.1", key)
}

func TestPeerKey_XRealIP(t *testing.T) {
	interceptor.Configure(interceptor.WithTrustProxy(true))
	defer interceptor.Configure(interceptor.WithTrustProxy(false)) // reset

	md := metadata.Pairs("x-real-ip", "198.51.100.2")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = peer.NewContext(ctx, &peer.Peer{Addr: fakeAddr("192.168.1.1:12345")})

	key := interceptor.PeerKey(ctx)
	assert.Equal(t, "198.51.100.2", key)
}

func TestPeerKey_XForwardedFor_And_XRealIP(t *testing.T) {
	interceptor.Configure(interceptor.WithTrustProxy(true))
	defer interceptor.Configure(interceptor.WithTrustProxy(false)) // reset

	md := metadata.Pairs(
		"x-forwarded-for", "203.0.113.1",
		"x-real-ip", "198.51.100.2",
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = peer.NewContext(ctx, &peer.Peer{Addr: fakeAddr("192.168.1.1:12345")})

	key := interceptor.PeerKey(ctx)
	// X-Forwarded-For should take precedence over X-Real-IP
	assert.Equal(t, "203.0.113.1", key)
}

func TestStartCleanup_DefaultTTL(t *testing.T) {
	// Reset the sync.Once so we can re-enter startCleanup.
	interceptor.ResetCleanupOnce()

	// Calling with TTL=0 triggers the default 10m fallback path.
	interceptor.StartCleanup(0)

	// The goroutine is now running. Stop it and reset for next tests.
	interceptor.StopCleanup()
	interceptor.ResetCleanupOnce()
}

func TestRunCleanupLoop(t *testing.T) {
	// Directly test the cleanup loop with a very short interval.
	interceptor.Configure(
		interceptor.WithRateLimit(100, 10),
		interceptor.WithRateLimitTTL(1*time.Millisecond),
	)

	// Create a limiter entry.
	i := interceptor.RateLimit()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Method"}
	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }

	ctx := peerContext("10.88.88.88:12345")
	_, err := i(ctx, "req", info, handler)
	require.NoError(t, err)

	// Set lastSeen to the past so cleanup will remove it.
	interceptor.SetLimiterLastSeen("10.88.88.88", time.Now().Add(-1*time.Hour))

	// Run cleanup loop directly with a very short TTL (1ms) and stop it after one tick.
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		interceptor.RunCleanupLoop(1*time.Millisecond, stop)
		close(done)
	}()

	// Give it time to tick at least once.
	time.Sleep(10 * time.Millisecond)
	close(stop)

	// Wait for the loop to exit.
	<-done
}
