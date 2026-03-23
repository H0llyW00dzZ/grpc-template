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

func TestRequestID_GeneratesID(t *testing.T) {
	i := interceptor.RequestID()

	var capturedCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Method"}

	resp, err := i(context.Background(), "req", info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "ok" {
		t.Errorf("got %v, want %q", resp, "ok")
	}

	id := interceptor.RequestIDFromContext(capturedCtx)
	if id == "" {
		t.Fatal("expected generated request ID, got empty string")
	}
	if len(id) < 32 {
		t.Errorf("request ID %q looks too short for a UUID", id)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	id := interceptor.RequestIDFromContext(capturedCtx)
	if id != "my-trace-id-123" {
		t.Errorf("got request ID %q, want %q", id, "my-trace-id-123")
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	id := interceptor.RequestIDFromContext(capturedCtx)
	if id == "" {
		t.Fatal("expected generated request ID in stream context, got empty string")
	}
}

func TestRequestIDFromContext_Empty(t *testing.T) {
	id := interceptor.RequestIDFromContext(context.Background())
	if id != "" {
		t.Errorf("expected empty string, got %q", id)
	}
}
