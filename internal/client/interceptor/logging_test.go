// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor_test

import (
	"context"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/client/interceptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLogging_Success(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Logging()

	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return nil
	}

	err := i(context.Background(), "/test.v1.Service/Method", nil, nil, nil, invoker)
	assert.NoError(t, err)
}

func TestLogging_Error(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Logging()

	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return status.Error(codes.NotFound, "not found")
	}

	err := i(context.Background(), "/test.v1.Service/Method", nil, nil, nil, invoker)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestStreamLogging_Success(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.StreamLogging()

	streamer := func(_ context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		return &mockClientStream{}, nil
	}

	cs, err := i(context.Background(), &grpc.StreamDesc{}, nil, "/test.v1.Service/Stream", streamer)
	require.NoError(t, err)
	require.NotNil(t, cs)
}

func TestStreamLogging_Error(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.StreamLogging()

	streamer := func(_ context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		return nil, status.Error(codes.Unavailable, "server unavailable")
	}

	cs, err := i(context.Background(), &grpc.StreamDesc{}, nil, "/test.v1.Service/Stream", streamer)
	require.Error(t, err)
	assert.Nil(t, cs)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, st.Code())
}
