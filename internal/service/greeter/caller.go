// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package greeter

import (
	"context"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	pb "github.com/H0llyW00dzZ/grpc-template/pkg/gen/helloworld/v1"
	"google.golang.org/grpc"
)

// Caller wraps the generated GreeterService client with logging and
// a simplified API. It is the client-side counterpart of [Service].
//
//	caller := greeter.NewCaller(c.Conn(), c.Logger())
//	reply, err := caller.SayHello(ctx, "World")
type Caller struct {
	client pb.GreeterServiceClient
	log    logging.Handler
}

// NewCaller creates a new Caller from the given client connection.
// Use [grpc.ClientConnInterface] so callers can pass either a
// [*grpc.ClientConn] or a test stub.
func NewCaller(conn grpc.ClientConnInterface, l logging.Handler) *Caller {
	return &Caller{
		client: pb.NewGreeterServiceClient(conn),
		log:    l,
	}
}

// SayHello sends a unary greeting request and returns the response.
// Optional [grpc.CallOption] values (e.g., [grpc.Header]) are forwarded
// to the underlying gRPC call.
func (c *Caller) SayHello(ctx context.Context, name string, opts ...grpc.CallOption) (*pb.SayHelloResponse, error) {
	c.log.Info("calling SayHello", "name", name)
	return c.client.SayHello(ctx, &pb.SayHelloRequest{Name: name}, opts...)
}

// SayHelloServerStream opens a server-streaming greeting and returns
// the stream for the caller to consume.
// Optional [grpc.CallOption] values are forwarded to the underlying
// gRPC call.
func (c *Caller) SayHelloServerStream(ctx context.Context, name string, opts ...grpc.CallOption) (pb.GreeterService_SayHelloServerStreamClient, error) {
	c.log.Info("calling SayHelloServerStream", "name", name)
	return c.client.SayHelloServerStream(ctx, &pb.SayHelloServerStreamRequest{Name: name}, opts...)
}
