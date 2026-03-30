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
	"google.golang.org/grpc"
)

func TestTimeout_SetsDeadline(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithDefaultTimeout(5 * time.Second))
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Timeout()

	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		_, ok := ctx.Deadline()
		assert.True(t, ok, "context should have a deadline")
		return nil
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	assert.NoError(t, err)
}

func TestTimeout_PreservesExistingDeadline(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(interceptor.WithDefaultTimeout(5 * time.Second))
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Timeout()

	originalDeadline := time.Now().Add(10 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), originalDeadline)
	defer cancel()

	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		// Original deadline should be preserved (not overwritten).
		assert.Equal(t, originalDeadline.Round(time.Millisecond), deadline.Round(time.Millisecond))
		return nil
	}

	err := i(ctx, "/test/Method", nil, nil, nil, invoker)
	assert.NoError(t, err)
}

func TestTimeout_NoConfig(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Timeout()

	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		_, ok := ctx.Deadline()
		assert.False(t, ok, "context should not have a deadline when no timeout configured")
		return nil
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	assert.NoError(t, err)
}
