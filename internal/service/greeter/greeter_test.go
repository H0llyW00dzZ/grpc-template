// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package greeter_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/client"
	_ "github.com/H0llyW00dzZ/grpc-template/internal/client/balancer"
	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/H0llyW00dzZ/grpc-template/internal/service/greeter"
	"github.com/H0llyW00dzZ/grpc-template/internal/testutil"
	pb "github.com/H0llyW00dzZ/grpc-template/pkg/gen/helloworld/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// testTimeout is the maximum time any single greeter test should take.
const testTimeout = 10 * time.Second

func startGreeterServer(t *testing.T) *bufconn.Listener {
	t.Helper()
	lis := testutil.NewBufListener()
	srv := grpc.NewServer()
	svc := greeter.NewService(logging.Default())
	svc.Register(srv)

	// Capture serve errors via a channel so we never call t.Errorf
	// from a goroutine that may outlive the test function.
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(lis)
	}()
	t.Cleanup(func() {
		srv.GracefulStop()
		// Drain errCh to ensure the goroutine has exited.
		<-errCh
	})
	return lis
}

func newGreeterClient(t *testing.T, lis *bufconn.Listener) pb.GreeterServiceClient {
	t.Helper()
	ctx := context.Background()
	conn, err := testutil.DialBufNet(ctx, lis)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return pb.NewGreeterServiceClient(conn)
}

func TestSayHello(t *testing.T) {
	lis := startGreeterServer(t)
	client := newGreeterClient(t, lis)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resp, err := client.SayHello(ctx, &pb.SayHelloRequest{Name: "World"})
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", resp.GetMessage())
}

func TestSayHello_EmptyName(t *testing.T) {
	lis := startGreeterServer(t)
	client := newGreeterClient(t, lis)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resp, err := client.SayHello(ctx, &pb.SayHelloRequest{Name: ""})
	require.NoError(t, err)
	assert.Equal(t, "Hello, !", resp.GetMessage())
}

func TestSayHelloServerStream(t *testing.T) {
	lis := startGreeterServer(t)
	client := newGreeterClient(t, lis)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	stream, err := client.SayHelloServerStream(ctx, &pb.SayHelloServerStreamRequest{Name: "Alice"})
	require.NoError(t, err)

	want := []string{
		"Hello, Alice!",
		"How are you, Alice?",
		"Good to see you, Alice!",
	}

	var got []string
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		got = append(got, resp.GetMessage())
	}

	require.Len(t, got, len(want))
	assert.Equal(t, want, got)
}

func TestSayHelloServerStream_ClientCancel(t *testing.T) {
	lis := startGreeterServer(t)
	client := newGreeterClient(t, lis)

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := client.SayHelloServerStream(ctx, &pb.SayHelloServerStreamRequest{Name: "Cancel"})
	require.NoError(t, err)

	// Receive first message successfully.
	_, err = stream.Recv()
	require.NoError(t, err)

	// Cancel context to force stream.Send() to fail on the server.
	cancel()

	// Subsequent reads must fail with Canceled.
	_, err = stream.Recv()
	require.Error(t, err)
	assert.Equal(t, codes.Canceled, status.Code(err))
}

func TestLoadBalancing_RoundRobin(t *testing.T) {
	lis := startGreeterServer(t)

	c := client.New("passthrough:///bufconn",
		client.WithInsecure(),
		client.WithLoadBalancing("round_robin"),
		client.WithDialOptions(
			grpc.WithContextDialer(testutil.BufDialer(lis)),
		),
	)
	require.NoError(t, c.Connect(context.Background()))
	t.Cleanup(func() { c.Close() })

	caller := greeter.NewCaller(c.Conn(), logging.Default())

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resp, err := caller.SayHello(ctx, "LoadBalanced")
	require.NoError(t, err)
	assert.Equal(t, "Hello, LoadBalanced!", resp.GetMessage())
}
