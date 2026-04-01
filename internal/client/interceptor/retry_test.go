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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRetry_NoConfig(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Retry()

	calls := 0
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		calls++
		return nil
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	assert.NoError(t, err)
	assert.Equal(t, 1, calls, "should invoke exactly once when retry not configured")
}

func TestRetry_SuccessNoRetry(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithRetry(3, 10*time.Millisecond))
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Retry()

	calls := 0
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		calls++
		return nil
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	assert.NoError(t, err)
	assert.Equal(t, 1, calls, "should not retry on success")
}

func TestRetry_RetryableError(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithRetry(2, 10*time.Millisecond))
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Retry()

	calls := 0
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		calls++
		if calls < 3 {
			return status.Error(codes.Unavailable, "temporarily unavailable")
		}
		return nil
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	assert.NoError(t, err)
	assert.Equal(t, 3, calls, "should retry twice and succeed on third attempt")
}

func TestRetry_NonRetryableError(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithRetry(3, 10*time.Millisecond))
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Retry()

	calls := 0
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		calls++
		return status.Error(codes.InvalidArgument, "bad request")
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	require.Error(t, err)
	assert.Equal(t, 1, calls, "should not retry non-retryable errors")

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestRetry_ExhaustsRetries(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithRetry(2, 10*time.Millisecond))
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Retry()

	calls := 0
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		calls++
		return status.Error(codes.Unavailable, "always unavailable")
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	require.Error(t, err)
	assert.Equal(t, 3, calls, "should try 1 initial + 2 retries")

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, st.Code())
}

func TestRetry_ContextCancelled(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithRetry(10, 5*time.Second))
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Retry()

	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		calls++
		// Cancel after first call to trigger context cancellation during backoff.
		cancel()
		return status.Error(codes.Unavailable, "unavailable")
	}

	err := i(ctx, "/test/Method", nil, nil, nil, invoker)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, 1, calls)
}

func TestRetry_ResourceExhausted(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithRetry(1, 10*time.Millisecond))
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Retry()

	calls := 0
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		calls++
		if calls == 1 {
			return status.Error(codes.ResourceExhausted, "rate limited")
		}
		return nil
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	assert.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestRetry_Aborted(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithRetry(1, 10*time.Millisecond))
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Retry()

	calls := 0
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		calls++
		if calls == 1 {
			return status.Error(codes.Aborted, "aborted")
		}
		return nil
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	assert.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestRetry_NonStatusError(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithRetry(3, 10*time.Millisecond))
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Retry()

	calls := 0
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		calls++
		return assert.AnError
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	require.Error(t, err)
	assert.Equal(t, 1, calls, "non-status errors should not be retried")
}

func TestBackoffDuration(t *testing.T) {
	d := interceptor.BackoffDuration(0, time.Second)
	assert.GreaterOrEqual(t, d, 500*time.Millisecond)
	assert.LessOrEqual(t, d, time.Second)
}

func TestBackoffDuration_ZeroBase(t *testing.T) {
	// halfBackoff = 0, should return expBackoff (0) without calling rand.
	d := interceptor.BackoffDuration(0, 0)
	assert.Equal(t, time.Duration(0), d)
}

func TestBackoffDuration_TinyBase(t *testing.T) {
	// With a 1ns base, halfBackoff = 0 after division, hits the guard.
	d := interceptor.BackoffDuration(0, time.Nanosecond)
	assert.GreaterOrEqual(t, d, time.Duration(0))
}

func TestBackoffDuration_OverflowCap(t *testing.T) {
	// Attempt 100 would overflow int64 without the cap at 62.
	// Verify it returns a positive duration instead of wrapping negative.
	d := interceptor.BackoffDuration(100, time.Second)
	assert.Greater(t, d, time.Duration(0), "overflow should be capped to a positive value")
}
