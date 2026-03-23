// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor_test

import (
	"context"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/H0llyW00dzZ/grpc-template/internal/server/interceptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecovery(t *testing.T) {
	i := interceptor.Recovery(logging.Default())

	handler := func(ctx context.Context, req any) (any, error) {
		panic("test panic")
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/PanicMethod"}

	resp, err := i(context.Background(), "request", info, handler)
	require.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestRecovery_NoPanic(t *testing.T) {
	i := interceptor.Recovery(logging.Default())

	handler := func(ctx context.Context, req any) (any, error) {
		return "safe", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/SafeMethod"}

	resp, err := i(context.Background(), "request", info, handler)
	require.NoError(t, err)
	assert.Equal(t, "safe", resp)
}

func TestStreamRecovery(t *testing.T) {
	i := interceptor.StreamRecovery(logging.Default())

	handler := func(srv any, stream grpc.ServerStream) error {
		panic("test stream panic")
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/PanicStream"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := i(nil, ss, info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestStreamRecovery_NoPanic(t *testing.T) {
	i := interceptor.StreamRecovery(logging.Default())

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/SafeStream"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := i(nil, ss, info, handler)
	require.NoError(t, err)
}
