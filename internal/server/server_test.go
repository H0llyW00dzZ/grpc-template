// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package server_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/server"
	"github.com/H0llyW00dzZ/grpc-template/internal/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
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

func TestWithTLS(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _ := generateTestCert(t, dir)

	// Should not panic with valid cert/key.
	srv := server.New(server.WithTLS(certFile, keyFile), server.WithPort("0"))
	if srv == nil {
		t.Fatal("New() with TLS returned nil")
	}

	// Verify the server can actually start with TLS.
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("Run with TLS: %v", err)
	}
}

func TestWithTLS_InvalidFiles(t *testing.T) {
	assertPanics(t, "WithTLS bad paths", func() {
		server.New(server.WithTLS("/no/such/cert.pem", "/no/such/key.pem"))
	})
}

func TestWithMutualTLS(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, caCertFile := generateTestCert(t, dir)

	srv := server.New(server.WithMutualTLS(certFile, keyFile, caCertFile), server.WithPort("0"))
	if srv == nil {
		t.Fatal("New() with mTLS returned nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("Run with mTLS: %v", err)
	}
}

func TestWithMutualTLS_InvalidCert(t *testing.T) {
	assertPanics(t, "WithMutualTLS bad cert", func() {
		server.New(server.WithMutualTLS("/bad/cert.pem", "/bad/key.pem", "/bad/ca.pem"))
	})
}

func TestWithMutualTLS_InvalidCA(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _ := generateTestCert(t, dir)

	assertPanics(t, "WithMutualTLS bad CA path", func() {
		server.New(server.WithMutualTLS(certFile, keyFile, "/no/such/ca.pem"))
	})
}

func TestWithMutualTLS_InvalidCAPEM(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _ := generateTestCert(t, dir)

	// Write a file that exists but is not valid PEM.
	badCA := filepath.Join(dir, "bad_ca.pem")
	if err := os.WriteFile(badCA, []byte("not a PEM"), 0o600); err != nil {
		t.Fatalf("write bad CA: %v", err)
	}

	assertPanics(t, "WithMutualTLS invalid PEM", func() {
		server.New(server.WithMutualTLS(certFile, keyFile, badCA))
	})
}

func TestWithKeepalive(t *testing.T) {
	srv := server.New(
		server.WithKeepalive(
			keepalive.ServerParameters{MaxConnectionIdle: 30 * time.Second},
			keepalive.EnforcementPolicy{MinTime: 10 * time.Second},
		),
		server.WithPort("0"),
	)
	if srv == nil {
		t.Fatal("New() with keepalive returned nil")
	}
}

func TestWithMaxMsgSize(t *testing.T) {
	srv := server.New(server.WithMaxMsgSize(8*1024*1024), server.WithPort("0"))
	if srv == nil {
		t.Fatal("New() with max msg size returned nil")
	}
}

func TestWithMaxConcurrentStreams(t *testing.T) {
	srv := server.New(server.WithMaxConcurrentStreams(100), server.WithPort("0"))
	if srv == nil {
		t.Fatal("New() with max concurrent streams returned nil")
	}
}

func TestWithGrpcOptions(t *testing.T) {
	srv := server.New(
		server.WithGrpcOptions(grpc.MaxRecvMsgSize(1024)),
		server.WithPort("0"),
	)
	if srv == nil {
		t.Fatal("New() with raw grpc options returned nil")
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

func TestStreamLoggingInterceptor(t *testing.T) {
	interceptor := server.StreamLoggingInterceptor()

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/TestStream"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := interceptor(nil, ss, info, handler)
	if err != nil {
		t.Fatalf("StreamLoggingInterceptor: unexpected error: %v", err)
	}
}

func TestStreamLoggingInterceptor_Error(t *testing.T) {
	interceptor := server.StreamLoggingInterceptor()

	handler := func(srv any, stream grpc.ServerStream) error {
		return status.Error(codes.Internal, "stream error")
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/TestStream"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := interceptor(nil, ss, info, handler)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.Internal {
		t.Errorf("got status %v, want Internal", err)
	}
}

func TestStreamRecoveryInterceptor(t *testing.T) {
	interceptor := server.StreamRecoveryInterceptor()

	handler := func(srv any, stream grpc.ServerStream) error {
		panic("test stream panic")
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/PanicStream"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := interceptor(nil, ss, info, handler)
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

func TestStreamRecoveryInterceptor_NoPanic(t *testing.T) {
	interceptor := server.StreamRecoveryInterceptor()

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.TestService/SafeStream"}
	ss := &fakeServerStream{ctx: context.Background()}

	err := interceptor(nil, ss, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterService(t *testing.T) {
	called := false
	registrar := func(gs *grpc.Server) {
		called = true
	}

	srv := server.New(server.WithPort("0"))
	srv.RegisterService(registrar)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !called {
		t.Error("RegisterService registrar was never called")
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

func TestServer_RunWithAllOptions(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _ := generateTestCert(t, dir)

	registrarCalled := false

	srv := server.New(
		server.WithPort("0"),
		server.WithTLS(certFile, keyFile),
		server.WithReflection(),
		server.WithUnaryInterceptors(server.LoggingInterceptor(), server.RecoveryInterceptor()),
		server.WithStreamInterceptors(func(
			srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler,
		) error {
			return handler(srv, ss)
		}),
		server.WithKeepalive(
			keepalive.ServerParameters{MaxConnectionIdle: 30 * time.Second},
			keepalive.EnforcementPolicy{MinTime: 10 * time.Second},
		),
		server.WithMaxMsgSize(8*1024*1024),
		server.WithMaxConcurrentStreams(100),
		server.WithGrpcOptions(),
	)
	srv.RegisterService(func(gs *grpc.Server) {
		registrarCalled = true
	})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("Run with all options: %v", err)
	}
	if !registrarCalled {
		t.Error("registrar was not called")
	}
}

func TestServer_RunInvalidPort(t *testing.T) {
	srv := server.New(server.WithPort("invalid-port"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := srv.Run(ctx)
	if err == nil {
		t.Fatal("expected error for invalid port, got nil")
	}
}

func TestServer_ServeError(t *testing.T) {
	// Create a listener and immediately close it.
	// Passing a closed listener via WithListener causes grpc.Server.Serve
	// to fail with an accept error, exercising the errCh path in Run.
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("create listener: %v", err)
	}
	lis.Close()

	srv := server.New(server.WithListener(lis))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = srv.Run(ctx)
	if err == nil {
		t.Fatal("expected serve error with closed listener, got nil")
	}
}

func TestWithListener(t *testing.T) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("create listener: %v", err)
	}

	srv := server.New(server.WithListener(lis))
	if srv == nil {
		t.Fatal("New() with listener returned nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("Run with listener: %v", err)
	}
}
