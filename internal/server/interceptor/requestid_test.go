// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor_test

import (
	"context"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/server/interceptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestRequestID_GeneratesID(t *testing.T) {
	i := interceptor.RequestID()

	var capturedCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Method"}

	resp, err := i(context.Background(), "req", info, handler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	id := interceptor.RequestIDFromContext(capturedCtx)
	require.NotEmpty(t, id)
	assert.GreaterOrEqual(t, len(id), 32, "request ID %q looks too short for a UUID", id)
}

func TestRequestID_PreservesExisting(t *testing.T) {
	i := interceptor.RequestID()

	var capturedCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Method"}

	md := metadata.Pairs("x-request-id", "my-trace-id-123")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := i(ctx, "req", info, handler)
	require.NoError(t, err)
	assert.Equal(t, "my-trace-id-123", interceptor.RequestIDFromContext(capturedCtx))
}

func TestStreamRequestID(t *testing.T) {
	i := interceptor.StreamRequestID()

	var capturedCtx context.Context
	handler := func(srv any, stream grpc.ServerStream) error {
		capturedCtx = stream.Context()
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.Svc/StreamMethod"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := i(nil, ss, info, handler)
	require.NoError(t, err)
	assert.NotEmpty(t, interceptor.RequestIDFromContext(capturedCtx))
}

func TestRequestIDFromContext_Empty(t *testing.T) {
	id := interceptor.RequestIDFromContext(context.Background())
	assert.Empty(t, id)
}
