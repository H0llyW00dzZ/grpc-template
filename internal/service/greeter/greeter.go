// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pb "github.com/H0llyW00dzZ/grpc-template/pkg/gen/helloworld/v1"
	"google.golang.org/grpc"
)

// GreeterService implements the Greeter gRPC service.
type GreeterService struct {
	pb.UnimplementedGreeterServer
}

// NewGreeterService returns a new GreeterService.
func NewGreeterService() *GreeterService {
	return &GreeterService{}
}

// Register registers the GreeterService on the given gRPC server.
// This satisfies the server.ServiceRegistrar function signature.
func (s *GreeterService) Register(srv *grpc.Server) {
	pb.RegisterGreeterServer(srv, s)
}

// SayHello handles a unary RPC and returns a greeting.
func (s *GreeterService) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	slog.Info("received SayHello request", "name", req.GetName())

	return &pb.HelloReply{
		Message: fmt.Sprintf("Hello, %s!", req.GetName()),
	}, nil
}

// SayHelloServerStream handles a server-streaming RPC by sending multiple greetings.
func (s *GreeterService) SayHelloServerStream(req *pb.HelloRequest, stream pb.Greeter_SayHelloServerStreamServer) error {
	slog.Info("received SayHelloServerStream request", "name", req.GetName())

	greetings := []string{
		fmt.Sprintf("Hello, %s!", req.GetName()),
		fmt.Sprintf("How are you, %s?", req.GetName()),
		fmt.Sprintf("Good to see you, %s!", req.GetName()),
	}

	for _, greeting := range greetings {
		if err := stream.Send(&pb.HelloReply{Message: greeting}); err != nil {
			return fmt.Errorf("failed to send greeting: %w", err)
		}
		// Simulate some processing delay between messages.
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}
