// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package client_test

import (
	"context"
	"fmt"
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
	"google.golang.org/grpc/test/bufconn"
)

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
	)

	greeterSvc := greeter.NewService(l)
	srv.RegisterService(greeterSvc.Register)

	go func() {
		if err := srv.Run(ctx); err != nil {
			t.Log("server stopped:", err)
		}
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lis := startTestServer(t, ctx)

	c := newBufClient(lis, client.WithHealthWatch())
	err := c.Connect(context.Background())
	require.NoError(t, err)

	// Give health watch goroutine a moment to start.
	time.Sleep(100 * time.Millisecond)

	err = c.Close()
	assert.NoError(t, err)
}

func TestCloseWithHealthWatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lis := startTestServer(t, ctx)

	c := newBufClient(lis, client.WithHealthWatch())
	err := c.Connect(context.Background())
	require.NoError(t, err)

	// Give health watch goroutine a moment to start.
	time.Sleep(100 * time.Millisecond)

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// Give health watch time to start and receive initial status.
	time.Sleep(200 * time.Millisecond)

	// Shut down the server — this causes the health watch stream to end
	// with an error, exercising the stream error path in startHealthWatch.
	cancel()
	time.Sleep(200 * time.Millisecond)

	err = c.Close()
	assert.NoError(t, err)
}

func TestHealthWatch_NoHealthService(t *testing.T) {
	// Start a plain gRPC server without health service registration.
	lis := testutil.NewBufListener()
	srv := grpc.NewServer()
	go func() {
		if err := srv.Serve(lis); err != nil {
			t.Log("server exited:", err)
		}
	}()
	t.Cleanup(func() { srv.GracefulStop() })

	c := client.New("passthrough:///bufconn",
		client.WithDialOptions(
			grpc.WithContextDialer(testutil.BufDialer(lis)),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
		client.WithLogger(logging.Default()),
		client.WithHealthWatch(),
	)

	err := c.Connect(context.Background())
	require.NoError(t, err)

	// Give health watch time to attempt and fail.
	time.Sleep(200 * time.Millisecond)

	err = c.Close()
	assert.NoError(t, err)
}

func TestHealthWatch_ImmediateClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	// we close the raw *grpc.ClientConn (bypassing Client.Close) so
	// the health context is still active but the transport is broken.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lis := startTestServer(t, ctx)

	c := newBufClient(lis, client.WithHealthWatch())
	err := c.Connect(context.Background())
	require.NoError(t, err)

	// Let health watch start and receive the initial status.
	time.Sleep(200 * time.Millisecond)

	// Close the raw conn — this kills the transport without cancelling
	// the health watch context, causing stream.Recv to fail with a
	// transport error while ctx.Err() is still nil.
	rawConn := c.Conn()
	rawConn.Close()

	// Wait for the health goroutine to log the warning and exit.
	time.Sleep(200 * time.Millisecond)
}

func TestHealthWatch_WatchCtxCancelled(t *testing.T) {
	// To cover the Watch() error path where ctx.Err() != nil,
	// we close the client immediately after Connect so that the
	// health context is cancelled before Watch can complete.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lis := startTestServer(t, ctx)

	c := newBufClient(lis, client.WithHealthWatch())
	c.SetWatchFunc(func(_ context.Context, _ *grpc.ClientConn) (healthgrpc.Health_WatchClient, error) {
		return nil, fmt.Errorf("injected watch error")
	})

	err := c.Connect(context.Background())
	require.NoError(t, err)

	// Give the goroutine time to call the injected watch func and hit the warn path.
	time.Sleep(100 * time.Millisecond)

	err = c.Close()
	assert.NoError(t, err)
}
