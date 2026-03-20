// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package server_test

import (
	"context"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/server"
	"github.com/H0llyW00dzZ/grpc-template/internal/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --------------------------------------------------------------------------
// Functional Options
// --------------------------------------------------------------------------

func TestNewServer_DefaultPort(t *testing.T) {
	srv := server.New()
	if srv == nil {
		t.Fatal("New() returned nil")
	}
	// The server should be created successfully with default options.
	// Port is unexported, so we verify indirectly via a successful Run test.
}

func TestNewServer_WithOptions(t *testing.T) {
	srv := server.New(
		server.WithPort("9090"),
		server.WithReflection(),
		server.WithUnaryInterceptors(server.LoggingInterceptor()),
		server.WithStreamInterceptors(),
	)
	if srv == nil {
		t.Fatal("New() with options returned nil")
	}
}

// --------------------------------------------------------------------------
// Interceptors
// --------------------------------------------------------------------------

func TestLoggingInterceptor(t *testing.T) {
	interceptor := server.LoggingInterceptor()

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/TestMethod"}

	resp, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Fatalf("LoggingInterceptor: unexpected error: %v", err)
	}
	if resp != "ok" {
		t.Errorf("got %v, want %q", resp, "ok")
	}
}

func TestLoggingInterceptor_Error(t *testing.T) {
	interceptor := server.LoggingInterceptor()

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.NotFound, "not found")
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/TestMethod"}

	_, err := interceptor(context.Background(), "request", info, handler)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.NotFound {
		t.Errorf("got status %v, want NotFound", err)
	}
}

func TestRecoveryInterceptor(t *testing.T) {
	interceptor := server.RecoveryInterceptor()

	// Handler that panics.
	handler := func(ctx context.Context, req any) (any, error) {
		panic("test panic")
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/PanicMethod"}

	resp, err := interceptor(context.Background(), "request", info, handler)
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

func TestRecoveryInterceptor_NoPanic(t *testing.T) {
	interceptor := server.RecoveryInterceptor()

	handler := func(ctx context.Context, req any) (any, error) {
		return "safe", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.TestService/SafeMethod"}

	resp, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "safe" {
		t.Errorf("got %v, want %q", resp, "safe")
	}
}

// --------------------------------------------------------------------------
// Server Lifecycle (integration smoke test)
// --------------------------------------------------------------------------

func TestServer_RunAndShutdown(t *testing.T) {
	_ = testutil.NewBufListener() // ensure testutil compiles with server tests

	srv := server.New(server.WithPort("0"))

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(ctx)
	}()

	// Cancel immediately to trigger graceful shutdown.
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("Run: %v", err)
	}
}
