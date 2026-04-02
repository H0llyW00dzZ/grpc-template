// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package greeter_test

import (
	"context"
	"io"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/service/greeter"
	"github.com/H0llyW00dzZ/grpc-template/internal/testutil"
	pb "github.com/H0llyW00dzZ/grpc-template/pkg/gen/helloworld/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// noopLogger is a zero-cost logger for benchmarks that discards all output.
type noopLogger struct{}

func (*noopLogger) Debug(_ string, _ ...any) {}
func (*noopLogger) Info(_ string, _ ...any)  {}
func (*noopLogger) Warn(_ string, _ ...any)  {}
func (*noopLogger) Error(_ string, _ ...any) {}

// benchEnv holds the shared bufconn server and client for greeter benchmarks.
type benchEnv struct {
	client pb.GreeterServiceClient
	srv    *grpc.Server
	lis    *bufconn.Listener
}

func newBenchEnv(b *testing.B) *benchEnv {
	b.Helper()
	lis := testutil.NewBufListener()
	srv := grpc.NewServer()
	svc := greeter.NewService(&noopLogger{})
	svc.Register(srv)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(lis) }()

	conn, err := testutil.DialBufNet(context.Background(), lis)
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() {
		conn.Close()
		srv.GracefulStop()
		<-errCh
	})

	return &benchEnv{
		client: pb.NewGreeterServiceClient(conn),
		srv:    srv,
		lis:    lis,
	}
}

// BenchmarkSayHello measures the end-to-end unary RPC throughput over
// an in-memory bufconn transport (no network overhead).
func BenchmarkSayHello(b *testing.B) {
	env := newBenchEnv(b)
	req := &pb.SayHelloRequest{Name: "Bench"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := env.client.SayHello(context.Background(), req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSayHello_Parallel measures unary RPC throughput under concurrent
// load, stress-testing the bufconn transport and service handler.
func BenchmarkSayHello_Parallel(b *testing.B) {
	env := newBenchEnv(b)
	req := &pb.SayHelloRequest{Name: "Bench"}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := env.client.SayHello(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSayHelloServerStream measures the server-streaming RPC throughput
// including stream setup, message receipt, and teardown.
// Note: each iteration includes the 500ms delay between messages by design.
// To measure pure stream throughput without delay, run the _DrainOnly variant.
func BenchmarkSayHelloServerStream(b *testing.B) {
	env := newBenchEnv(b)
	req := &pb.SayHelloServerStreamRequest{Name: "Bench"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		stream, err := env.client.SayHelloServerStream(context.Background(), req)
		if err != nil {
			b.Fatal(err)
		}
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkNewService measures the cost of constructing a new greeter service.
func BenchmarkNewService(b *testing.B) {
	l := &noopLogger{}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = greeter.NewService(l)
	}
}
