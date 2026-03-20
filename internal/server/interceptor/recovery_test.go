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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecovery(t *testing.T) {
	i := interceptor.Recovery()

	handler := func(ctx context.Context, req any) (any, error) {
		panic("test panic")
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/PanicMethod"}

	resp, err := i(context.Background(), "request", info, handler)
	if err == nil {
		t.Fatal("expected error after panic, got nil")
	}
	if resp != nil {
		t.Errorf("expected nil response after panic, got %v", resp)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("got code %v, want Internal", st.Code())
	}
}

func TestRecovery_NoPanic(t *testing.T) {
	i := interceptor.Recovery()

	handler := func(ctx context.Context, req any) (any, error) {
		return "safe", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/SafeMethod"}

	resp, err := i(context.Background(), "request", info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "safe" {
		t.Errorf("got %v, want %q", resp, "safe")
	}
}

func TestStreamRecovery(t *testing.T) {
	i := interceptor.StreamRecovery()

	handler := func(srv any, stream grpc.ServerStream) error {
		panic("test stream panic")
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/PanicStream"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := i(nil, ss, info, handler)
	if err == nil {
		t.Fatal("expected error after panic, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("got code %v, want Internal", st.Code())
	}
}

func TestStreamRecovery_NoPanic(t *testing.T) {
	i := interceptor.StreamRecovery()

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/SafeStream"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := i(nil, ss, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
