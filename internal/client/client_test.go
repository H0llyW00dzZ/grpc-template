// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package client_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/client"
	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/H0llyW00dzZ/grpc-template/internal/server"
	"github.com/H0llyW00dzZ/grpc-template/internal/service/greeter"
	"github.com/H0llyW00dzZ/grpc-template/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/test/bufconn"
)

// errorWatchStream is a mock Health_WatchClient that returns an error on Recv.
type errorWatchStream struct {
	grpc.ClientStream
	err error
}

func (s *errorWatchStream) Recv() (*healthgrpc.HealthCheckResponse, error) {
	return nil, s.err
}

func TestNew(t *testing.T) {
	c := client.New("localhost:50051")
	require.NotNil(t, c)
	assert.NotNil(t, c.Logger())
}

func TestNewWithOptions(t *testing.T) {
	l := logging.Default()
	c := client.New("localhost:50051",
		client.WithInsecure(),
		client.WithLogger(l),
		client.WithHealthWatch(),
	)
	require.NotNil(t, c)
	assert.Equal(t, l, c.Logger())
}

func TestConnPanicsBeforeConnect(t *testing.T) {
	c := client.New("localhost:50051")
	assert.Panics(t, func() {
		c.Conn()
	})
}

func TestStateBeforeConnect(t *testing.T) {
	c := client.New("localhost:50051")
	assert.Equal(t, connectivity.Shutdown, c.State())
}

func TestCloseBeforeConnect(t *testing.T) {
	c := client.New("localhost:50051")
	err := c.Close()
	assert.NoError(t, err)
}

// startTestServer creates and starts a gRPC server with a greeter service
// on a bufconn listener. Returns the listener. The server shuts down when
// ctx is cancelled.
func startTestServer(t *testing.T, ctx context.Context) *bufconn.Listener {
	t.Helper()
	lis := testutil.NewBufListener()
	l := logging.Default()

	srv := server.New(
		server.WithListener(lis),
		server.WithLogger(l),
		server.WithReflection(),
	)

	greeterSvc := greeter.NewService(l)
	srv.RegisterService(greeterSvc.Register)

	go func() {
		_ = srv.Run(ctx)
	}()

	return lis
}

// newBufClient creates a client configured for bufconn testing.
func newBufClient(lis *bufconn.Listener, opts ...client.Option) *client.Client {
	defaults := []client.Option{
		client.WithDialOptions(
			grpc.WithContextDialer(testutil.BufDialer(lis)),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
		client.WithLogger(logging.Default()),
	}
	return client.New("passthrough:///bufconn", append(defaults, opts...)...)
}

func TestConnectAndClose(t *testing.T) {
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	c := newBufClient(lis)
	err := c.Connect(context.Background())
	require.NoError(t, err)

	conn := c.Conn()
	require.NotNil(t, conn)

	err = c.Close()
	assert.NoError(t, err)
}

func TestStateAfterConnect(t *testing.T) {
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	c := newBufClient(lis)
	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	// After connect, state should not be Shutdown.
	state := c.State()
	assert.NotEqual(t, connectivity.Shutdown, state)
}

func TestWaitReady(t *testing.T) {
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	c := newBufClient(lis)
	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	readyCtx, readyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readyCancel()

	err = c.WaitReady(readyCtx)
	assert.NoError(t, err)
	assert.Equal(t, connectivity.Ready, c.State())
}

func TestWaitReady_ContextExpired(t *testing.T) {
	// Connect to a non-existent server to ensure WaitReady times out.
	c := client.New("localhost:1",
		client.WithInsecure(),
		client.WithLogger(logging.Default()),
	)
	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	// Use a very short timeout so WaitReady fails fast.
	readyCtx, readyCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer readyCancel()

	err = c.WaitReady(readyCtx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context expired waiting for ready state")
}

func TestConnectWithHealthWatch(t *testing.T) {
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	c := newBufClient(lis, client.WithHealthWatch())
	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	// Wait for the connection to become ready, confirming the
	// health watch goroutine is running.
	require.Eventually(t, func() bool {
		return c.State() == connectivity.Ready
	}, 5*time.Second, 50*time.Millisecond, "connection should reach Ready state")
}

func TestCloseWithHealthWatch(t *testing.T) {
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	c := newBufClient(lis, client.WithHealthWatch())
	err := c.Connect(context.Background())
	require.NoError(t, err)

	// Wait for the health watch goroutine to be running.
	require.Eventually(t, func() bool {
		return c.State() == connectivity.Ready
	}, 5*time.Second, 50*time.Millisecond, "connection should reach Ready state")

	// Close should cancel health watch and not hang.
	err = c.Close()
	assert.NoError(t, err)

	// Double close should be safe.
	err = c.Close()
	assert.NoError(t, err)
}

func TestConnectWithTLSConfig(t *testing.T) {
	dir := t.TempDir()
	_, _, caCertFile := generateTestCert(t, dir)

	// Create a client with TLS (won't actually connect since no
	// TLS server is available, but exercises buildDialOpts TLS path).
	c := client.New("localhost:50051",
		client.WithTLS(caCertFile),
		client.WithLogger(logging.Default()),
	)

	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
}

func TestConnectDefault(t *testing.T) {
	// Neither insecure nor TLS — exercises the default branch in buildDialOpts.
	c := client.New("localhost:50051",
		client.WithLogger(logging.Default()),
	)
	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
}

func TestConnectWithInsecure(t *testing.T) {
	// Exercises the insecureCreds branch specifically in buildDialOpts.
	c := client.New("localhost:50051",
		client.WithInsecure(),
		client.WithLogger(logging.Default()),
	)
	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
}

func TestConnectWithInterceptors(t *testing.T) {
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	// Exercises the unary and stream interceptor chain branches in buildDialOpts.
	c := newBufClient(lis,
		client.WithUnaryInterceptors(
			func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
				return invoker(ctx, method, req, reply, cc, opts...)
			},
		),
		client.WithStreamInterceptors(
			func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
				return streamer(ctx, desc, cc, method, opts...)
			},
		),
	)

	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
}

func TestHealthWatch_ServerShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	lis := startTestServer(t, ctx)

	c := newBufClient(lis, client.WithHealthWatch())
	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	// Wait for health watch to start and receive initial status.
	require.Eventually(t, func() bool {
		return c.State() == connectivity.Ready
	}, 5*time.Second, 50*time.Millisecond, "connection should reach Ready state")

	// Shut down the server — this causes the health watch stream to end
	// with an error, exercising the stream error path in startHealthWatch.
	cancel()

	// Wait for the client to notice the server is gone.
	require.Eventually(t, func() bool {
		s := c.State()
		return s == connectivity.TransientFailure || s == connectivity.Idle
	}, 5*time.Second, 50*time.Millisecond, "client should detect server shutdown")
}

func TestHealthWatch_NoHealthService(t *testing.T) {
	// Start a plain gRPC server without health service registration.
	lis := testutil.NewBufListener()
	srv := grpc.NewServer()
	go func() {
		if err := srv.Serve(lis); err != nil {
			// Only log if test is still running.
			select {
			case <-t.Context().Done():
			default:
				t.Log("server exited:", err)
			}
		}
	}()
	t.Cleanup(func() { srv.GracefulStop() })

	var watchAttempts atomic.Int32
	c := client.New("passthrough:///bufconn",
		client.WithDialOptions(
			grpc.WithContextDialer(testutil.BufDialer(lis)),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
		client.WithLogger(logging.Default()),
		client.WithHealthWatch(),
	)
	// Wrap the real health watch to count attempts.
	c.SetWatchFunc(func(ctx context.Context, conn *grpc.ClientConn) (healthgrpc.Health_WatchClient, error) {
		watchAttempts.Add(1)
		return healthgrpc.NewHealthClient(conn).Watch(ctx, &healthgrpc.HealthCheckRequest{})
	})

	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	// Wait for the health watch to attempt at least once.
	require.Eventually(t, func() bool {
		return watchAttempts.Load() >= 1
	}, 5*time.Second, 50*time.Millisecond, "health watch should attempt to connect")
}

func TestHealthWatch_ImmediateClose(t *testing.T) {
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	c := newBufClient(lis, client.WithHealthWatch())
	err := c.Connect(context.Background())
	require.NoError(t, err)

	// Immediately close — cancels the health watch context before
	// the Watch RPC has a chance to establish. This exercises the
	// ctx.Err() != nil early-return guard in startHealthWatch.
	err = c.Close()
	assert.NoError(t, err)
}

func TestConnect_DialError(t *testing.T) {
	c := client.New("localhost:50051",
		client.WithInsecure(),
		client.WithLogger(logging.Default()),
	)

	// Inject a dialer that always fails.
	c.SetDialFunc(func(_ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
		return nil, fmt.Errorf("injected dial error")
	})

	err := c.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create connection")
	assert.Contains(t, err.Error(), "injected dial error")
}

func TestHealthWatch_WatchRPCError(t *testing.T) {
	// To cover the stream.Recv() error path where ctx.Err() == nil,
	// we inject a watch function that returns a stream which fails on
	// the second Recv call, simulating a broken transport while the
	// health context is still active.
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	var recvCalls atomic.Int32
	c := newBufClient(lis, client.WithHealthWatch())
	c.SetWatchFunc(func(wctx context.Context, conn *grpc.ClientConn) (healthgrpc.Health_WatchClient, error) {
		// First call succeeds (returns a stream that errors on Recv),
		// covering the stream.Recv error path.
		recvCalls.Add(1)
		return &errorWatchStream{err: fmt.Errorf("transport broken")}, nil
	})

	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	// Wait for the health goroutine to hit the Recv error and retry.
	require.Eventually(t, func() bool {
		return recvCalls.Load() >= 2
	}, 5*time.Second, 50*time.Millisecond, "health watch should retry after stream Recv error")
}

func TestHealthWatch_WatchCtxCancelled(t *testing.T) {
	// To cover the Watch() error path where ctx.Err() != nil,
	// we close the client immediately after Connect so that the
	// health context is cancelled before Watch can complete.
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	c := newBufClient(lis, client.WithHealthWatch())
	err := c.Connect(context.Background())
	require.NoError(t, err)

	// Immediately close — cancels the health context before Watch completes.
	err = c.Close()
	assert.NoError(t, err)
}

func TestHealthWatch_WatchFuncError(t *testing.T) {
	// Inject a watch function that returns an error to cover the Watch()
	// error path where ctx.Err() == nil. This path is unreachable with
	// the real gRPC client since Watch always defers errors to Recv.
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	var calls atomic.Int32
	c := newBufClient(lis, client.WithHealthWatch())
	c.SetWatchFunc(func(_ context.Context, _ *grpc.ClientConn) (healthgrpc.Health_WatchClient, error) {
		calls.Add(1)
		return nil, fmt.Errorf("injected watch error")
	})

	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	// Wait for the goroutine to call the injected watch func.
	require.Eventually(t, func() bool {
		return calls.Load() >= 1
	}, 5*time.Second, 50*time.Millisecond, "health watch should call injected func")
}

func TestHealthWatch_RetriesAfterWatchError(t *testing.T) {
	// Verify that the health watch goroutine retries after the backoff
	// sleep completes (covers sleepOrDone time.After path and the
	// backoff-doubling continue in startHealthWatch).
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	var calls atomic.Int32
	c := newBufClient(lis, client.WithHealthWatch())
	c.SetWatchFunc(func(_ context.Context, _ *grpc.ClientConn) (healthgrpc.Health_WatchClient, error) {
		calls.Add(1)
		return nil, fmt.Errorf("injected watch error")
	})

	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	// Poll until the goroutine has retried at least once (after 500ms backoff).
	require.Eventually(t, func() bool {
		return calls.Load() >= 2
	}, 5*time.Second, 50*time.Millisecond, "health watch should retry after backoff")
}

func TestHealthWatch_ReconnectsAfterStreamEnd(t *testing.T) {
	// Verify that the health watch goroutine reconnects after a stream
	// error once the backoff sleep completes (covers the post-stream-error
	// sleep path and backoff doubling in startHealthWatch).
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	var calls atomic.Int32
	c := newBufClient(lis, client.WithHealthWatch())
	c.SetWatchFunc(func(_ context.Context, _ *grpc.ClientConn) (healthgrpc.Health_WatchClient, error) {
		calls.Add(1)
		return &errorWatchStream{err: fmt.Errorf("stream broken")}, nil
	})

	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	// Poll until the goroutine has reconnected at least once (after 500ms backoff).
	require.Eventually(t, func() bool {
		return calls.Load() >= 2
	}, 5*time.Second, 50*time.Millisecond, "health watch should reconnect after stream error")
}

func TestConnect_AlreadyConnected(t *testing.T) {
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	c := newBufClient(lis)
	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	// Second Connect should fail without leaking the first connection.
	err = c.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already connected")
}

func TestConnect_ReconnectAfterClose(t *testing.T) {
	ctx := t.Context()

	lis := startTestServer(t, ctx)

	c := newBufClient(lis)

	// First cycle: connect then close.
	err := c.Connect(context.Background())
	require.NoError(t, err)
	require.NoError(t, c.Close())

	// Second cycle: should succeed after Close cleared c.conn.
	err = c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	assert.NotEqual(t, connectivity.Shutdown, c.State())
}

func TestClient_ListServices(t *testing.T) {
	ctx := t.Context()
	lis := startTestServer(t, ctx)

	c := newBufClient(lis)
	require.NoError(t, c.Connect(ctx))
	t.Cleanup(func() { c.Close() })

	services, err := c.ListServices(ctx)
	require.NoError(t, err)
	require.Contains(t, services, "helloworld.v1.GreeterService")
	require.Contains(t, services, "grpc.reflection.v1.ServerReflection")
}

func TestClient_ListServices_NoReflection(t *testing.T) {
	// Start a server without reflection to exercise the Recv error path
	// (the server sends an error response when it doesn't support reflection).
	lis := testutil.NewBufListener()
	srv := grpc.NewServer()
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.GracefulStop() })

	c := newBufClient(lis)
	require.NoError(t, c.Connect(context.Background()))
	t.Cleanup(func() { c.Close() })

	_, err := c.ListServices(context.Background())
	require.Error(t, err)
}

func TestClient_ListServices_CancelledContext(t *testing.T) {
	ctx := t.Context()
	lis := startTestServer(t, ctx)

	c := newBufClient(lis)
	require.NoError(t, c.Connect(context.Background()))
	t.Cleanup(func() { c.Close() })

	// Use an already-cancelled context so the stream Send/Recv fails,
	// exercising the Send error path (the stream is created lazily so
	// ServerReflectionInfo itself succeeds even with a cancelled context).
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.ListServices(cancelledCtx)
	require.Error(t, err)
}

func TestClient_ListServices_SendError(t *testing.T) {
	ctx := t.Context()
	lis := startTestServer(t, ctx)

	c := newBufClient(lis)
	require.NoError(t, c.Connect(context.Background()))
	t.Cleanup(func() { c.Close() })

	// Inject a list function that returns an error to exercise
	// the error propagation path in ListServices.
	c.SetListFunc(func(_ context.Context, _ *grpc.ClientConn) ([]string, error) {
		return nil, fmt.Errorf("injected list error")
	})

	_, err := c.ListServices(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "injected list error")
}

// fakeReflectionServer returns an ErrorResponse instead of a ListServicesResponse,
// exercising the "unexpected response type" branch in defaultListServices.
type fakeReflectionServer struct {
	grpc_reflection_v1.UnimplementedServerReflectionServer
}

func (s *fakeReflectionServer) ServerReflectionInfo(stream grpc_reflection_v1.ServerReflection_ServerReflectionInfoServer) error {
	// Read the client's request.
	if _, err := stream.Recv(); err != nil {
		return err
	}
	// Reply with an error response — not ListServicesResponse.
	return stream.Send(&grpc_reflection_v1.ServerReflectionResponse{
		MessageResponse: &grpc_reflection_v1.ServerReflectionResponse_ErrorResponse{
			ErrorResponse: &grpc_reflection_v1.ErrorResponse{
				ErrorCode:    1,
				ErrorMessage: "fake",
			},
		},
	})
}

func TestDefaultListServices_UnexpectedResponse(t *testing.T) {
	lis := testutil.NewBufListener()
	srv := grpc.NewServer()
	grpc_reflection_v1.RegisterServerReflectionServer(srv, &fakeReflectionServer{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.GracefulStop() })

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(testutil.BufDialer(lis)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	_, err = client.DefaultListServices(context.Background(), conn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response type")
}

func TestDefaultListServices_SendError(t *testing.T) {
	lis := testutil.NewBufListener()
	srv := grpc.NewServer()
	grpc_reflection_v1.RegisterServerReflectionServer(srv, &fakeReflectionServer{})
	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(testutil.BufDialer(lis)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	// Force the connection to become ready, then close it.
	// The closed connection makes Send fail.
	conn.Connect()
	require.Eventually(t, func() bool {
		return conn.GetState() == connectivity.Ready
	}, 3*time.Second, 10*time.Millisecond)
	conn.Close()

	// Stop the server too so the transport is fully dead.
	srv.Stop()

	_, err = client.DefaultListServices(context.Background(), conn)
	require.Error(t, err)
}
