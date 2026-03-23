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

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/H0llyW00dzZ/grpc-template/internal/server"
	"github.com/H0llyW00dzZ/grpc-template/internal/server/interceptor"
	"github.com/H0llyW00dzZ/grpc-template/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func TestNewServer_DefaultPort(t *testing.T) {
	srv := server.New()
	require.NotNil(t, srv)
}

func TestNewServer_WithOptions(t *testing.T) {
	srv := server.New(
		server.WithPort("9090"),
		server.WithReflection(),
		server.WithUnaryInterceptors(interceptor.Logging(logging.Default())),
		server.WithStreamInterceptors(),
	)
	require.NotNil(t, srv)
}

func TestWithTLS(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _ := generateTestCert(t, dir)

	srv := server.New(server.WithTLS(certFile, keyFile), server.WithPort("0"))
	require.NotNil(t, srv)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()
	cancel()

	require.NoError(t, <-errCh)
}

func TestWithTLS_InvalidFiles(t *testing.T) {
	require.Panics(t, func() {
		server.New(server.WithTLS("/no/such/cert.pem", "/no/such/key.pem"))
	})
}

func TestWithMutualTLS(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, caCertFile := generateTestCert(t, dir)

	srv := server.New(server.WithMutualTLS(certFile, keyFile, caCertFile), server.WithPort("0"))
	require.NotNil(t, srv)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()
	cancel()

	require.NoError(t, <-errCh)
}

func TestWithMutualTLS_InvalidCert(t *testing.T) {
	require.Panics(t, func() {
		server.New(server.WithMutualTLS("/bad/cert.pem", "/bad/key.pem", "/bad/ca.pem"))
	})
}

func TestWithMutualTLS_InvalidCA(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _ := generateTestCert(t, dir)

	require.Panics(t, func() {
		server.New(server.WithMutualTLS(certFile, keyFile, "/no/such/ca.pem"))
	})
}

func TestWithMutualTLS_InvalidCAPEM(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _ := generateTestCert(t, dir)

	badCA := filepath.Join(dir, "bad_ca.pem")
	require.NoError(t, os.WriteFile(badCA, []byte("not a PEM"), 0o600))

	require.Panics(t, func() {
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
	require.NotNil(t, srv)
}

func TestWithMaxMsgSize(t *testing.T) {
	srv := server.New(server.WithMaxMsgSize(8*1024*1024), server.WithPort("0"))
	require.NotNil(t, srv)
}

func TestWithMaxConcurrentStreams(t *testing.T) {
	srv := server.New(server.WithMaxConcurrentStreams(100), server.WithPort("0"))
	require.NotNil(t, srv)
}

func TestWithGrpcOptions(t *testing.T) {
	srv := server.New(
		server.WithGrpcOptions(grpc.MaxRecvMsgSize(1024)),
		server.WithPort("0"),
	)
	require.NotNil(t, srv)
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

	require.NoError(t, <-errCh)
	assert.True(t, called)
}

func TestServer_RunAndShutdown(t *testing.T) {
	_ = testutil.NewBufListener()

	srv := server.New(server.WithPort("0"))

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(ctx)
	}()

	cancel()

	require.NoError(t, <-errCh)
}

func TestServer_RunWithAllOptions(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _ := generateTestCert(t, dir)

	registrarCalled := false

	srv := server.New(
		server.WithPort("0"),
		server.WithTLS(certFile, keyFile),
		server.WithReflection(),
		server.WithUnaryInterceptors(interceptor.Logging(logging.Default()), interceptor.Recovery(logging.Default())),
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

	require.NoError(t, <-errCh)
	assert.True(t, registrarCalled)
}

func TestServer_RunInvalidPort(t *testing.T) {
	srv := server.New(server.WithPort("invalid-port"))

	ctx := t.Context()

	err := srv.Run(ctx)
	require.Error(t, err)
}

func TestServer_ServeError(t *testing.T) {
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	lis.Close()

	srv := server.New(server.WithListener(lis))

	ctx := t.Context()

	err = srv.Run(ctx)
	require.Error(t, err)
}

func TestWithListener(t *testing.T) {
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	srv := server.New(server.WithListener(lis))
	require.NotNil(t, srv)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()
	cancel()

	require.NoError(t, <-errCh)
}
