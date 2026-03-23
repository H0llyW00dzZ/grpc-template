// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package greeter_test

import (
	"context"
	"io"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/H0llyW00dzZ/grpc-template/internal/service/greeter"
	"github.com/H0llyW00dzZ/grpc-template/internal/testutil"
	pb "github.com/H0llyW00dzZ/grpc-template/pkg/gen/helloworld/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func startGreeterServer(t *testing.T) *bufconn.Listener {
	t.Helper()
	lis := testutil.NewBufListener()
	srv := grpc.NewServer()
	svc := greeter.NewService(logging.Default())
	svc.Register(srv)
	go func() {
		if err := srv.Serve(lis); err != nil {
			t.Errorf("server exited with error: %v", err)
		}
	}()
	t.Cleanup(func() { srv.GracefulStop() })
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

	resp, err := client.SayHello(context.Background(), &pb.SayHelloRequest{Name: "World"})
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", resp.GetMessage())
}

func TestSayHello_EmptyName(t *testing.T) {
	lis := startGreeterServer(t)
	client := newGreeterClient(t, lis)

	resp, err := client.SayHello(context.Background(), &pb.SayHelloRequest{Name: ""})
	require.NoError(t, err)
	assert.Equal(t, "Hello, !", resp.GetMessage())
}

func TestSayHelloServerStream(t *testing.T) {
	lis := startGreeterServer(t)
	client := newGreeterClient(t, lis)

	stream, err := client.SayHelloServerStream(context.Background(), &pb.SayHelloServerStreamRequest{Name: "Alice"})
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
