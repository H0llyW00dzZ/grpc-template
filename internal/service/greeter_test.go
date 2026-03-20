// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package service_test

import (
	"context"
	"io"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/service"
	"github.com/H0llyW00dzZ/grpc-template/internal/testutil"
	pb "github.com/H0llyW00dzZ/grpc-template/pkg/gen/helloworld/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// startGreeterServer starts a gRPC server with the GreeterService registered
// on an in-memory bufconn listener for testing.
func startGreeterServer(t *testing.T) *bufconn.Listener {
	t.Helper()
	lis := testutil.NewBufListener()
	srv := grpc.NewServer()
	svc := service.NewGreeterService()
	svc.Register(srv)
	go func() {
		if err := srv.Serve(lis); err != nil {
			t.Errorf("server exited with error: %v", err)
		}
	}()
	t.Cleanup(func() { srv.GracefulStop() })
	return lis
}

// newGreeterClient creates a GreeterClient connected to the given bufconn listener.
func newGreeterClient(t *testing.T, lis *bufconn.Listener) pb.GreeterClient {
	t.Helper()
	ctx := context.Background()
	conn, err := testutil.DialBufNet(ctx, lis)
	if err != nil {
		t.Fatalf("failed to dial bufconn: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return pb.NewGreeterClient(conn)
}

func TestSayHello(t *testing.T) {
	lis := startGreeterServer(t)
	client := newGreeterClient(t, lis)

	resp, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "World"})
	if err != nil {
		t.Fatalf("SayHello: %v", err)
	}
	want := "Hello, World!"
	if resp.GetMessage() != want {
		t.Errorf("got %q, want %q", resp.GetMessage(), want)
	}
}

func TestSayHello_EmptyName(t *testing.T) {
	lis := startGreeterServer(t)
	client := newGreeterClient(t, lis)

	resp, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: ""})
	if err != nil {
		t.Fatalf("SayHello: %v", err)
	}
	want := "Hello, !"
	if resp.GetMessage() != want {
		t.Errorf("got %q, want %q", resp.GetMessage(), want)
	}
}

func TestSayHelloServerStream(t *testing.T) {
	lis := startGreeterServer(t)
	client := newGreeterClient(t, lis)

	stream, err := client.SayHelloServerStream(context.Background(), &pb.HelloRequest{Name: "Alice"})
	if err != nil {
		t.Fatalf("SayHelloServerStream: %v", err)
	}

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
		if err != nil {
			t.Fatalf("stream.Recv: %v", err)
		}
		got = append(got, resp.GetMessage())
	}

	if len(got) != len(want) {
		t.Fatalf("received %d messages, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("message[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
