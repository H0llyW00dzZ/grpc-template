// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package greeter provides the Greeter gRPC service implementation.
// It handles unary and server-streaming RPCs defined in the helloworld/v1 proto.
package greeter

import (
	"context"
	"fmt"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	pb "github.com/H0llyW00dzZ/grpc-template/pkg/gen/helloworld/v1"
	"google.golang.org/grpc"
)

// Service implements the GreeterService gRPC service.
type Service struct {
	pb.UnimplementedGreeterServiceServer
	log logging.Handler
}

// NewService returns a new greeter Service using the provided logger.
func NewService(l logging.Handler) *Service {
	return &Service{log: l}
}

// Register registers the greeter Service on the given gRPC server.
// This satisfies the server.ServiceRegistrar function signature.
func (s *Service) Register(srv *grpc.Server) {
	pb.RegisterGreeterServiceServer(srv, s)
}

// SayHello handles a unary RPC and returns a greeting.
func (s *Service) SayHello(ctx context.Context, req *pb.SayHelloRequest) (*pb.SayHelloResponse, error) {
	s.log.Info("received SayHello request", "name", req.GetName())

	return &pb.SayHelloResponse{
		Message: fmt.Sprintf("Hello, %s!", req.GetName()),
	}, nil
}

// SayHelloServerStream handles a server-streaming RPC by sending multiple greetings.
func (s *Service) SayHelloServerStream(req *pb.SayHelloServerStreamRequest, stream pb.GreeterService_SayHelloServerStreamServer) error {
	s.log.Info("received SayHelloServerStream request", "name", req.GetName())

	greetings := []string{
		fmt.Sprintf("Hello, %s!", req.GetName()),
		fmt.Sprintf("How are you, %s?", req.GetName()),
		fmt.Sprintf("Good to see you, %s!", req.GetName()),
	}

	for _, greeting := range greetings {
		if err := stream.Send(&pb.SayHelloServerStreamResponse{Message: greeting}); err != nil {
			return fmt.Errorf("failed to send greeting: %w", err)
		}
		// Simulate some processing delay between messages, respecting
		// client cancellation so the server stops promptly instead of
		// sleeping the full duration. On cancellation, the next Send
		// will return the appropriate error.
		delay := time.NewTimer(500 * time.Millisecond)
		select {
		case <-delay.C:
		case <-stream.Context().Done():
			delay.Stop()
		}
	}

	return nil
}
