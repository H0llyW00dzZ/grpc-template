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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLogging(t *testing.T) {
	i := interceptor.Logging(logging.Default())

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/TestMethod"}

	resp, err := i(context.Background(), "request", info, handler)
	if err != nil {
		t.Fatalf("LoggingInterceptor: unexpected error: %v", err)
	}
	if resp != "ok" {
		t.Errorf("got %v, want %q", resp, "ok")
	}
}

func TestLogging_Error(t *testing.T) {
	i := interceptor.Logging(logging.Default())

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/TestMethod"}

	_, err := i(context.Background(), "request", info, handler)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.NotFound {
		t.Errorf("got status %v, want NotFound", err)
	}
}

func TestStreamLogging(t *testing.T) {
	i := interceptor.StreamLogging(logging.Default())

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/TestStream"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := i(nil, ss, info, handler)
	if err != nil {
		t.Fatalf("StreamLoggingInterceptor: unexpected error: %v", err)
	}
}

func TestStreamLogging_Error(t *testing.T) {
	i := interceptor.StreamLogging(logging.Default())

	handler := func(srv any, stream grpc.ServerStream) error {
		return status.Error(codes.Internal, "stream error")
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/TestStream"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := i(nil, ss, info, handler)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.Internal {
		t.Errorf("got status %v, want Internal", err)
	}
}
