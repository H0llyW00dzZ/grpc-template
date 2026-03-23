// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package main is the entry point for the gRPC client demo.
package main

import (
	"context"
	"io"
	"log"
	"log/slog"
	"time"

	pb "github.com/H0llyW00dzZ/grpc-template/pkg/gen/helloworld/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultAddr = "localhost:50051"
	defaultName = "World"
)

func main() {
	// Create a gRPC client connection.
	conn, err := grpc.NewClient(
		defaultAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewGreeterServiceClient(conn)

	// --- Unary RPC ---
	slog.Info("calling SayHello (unary)...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reply, err := client.SayHello(ctx, &pb.SayHelloRequest{Name: defaultName})
	if err != nil {
		log.Fatalf("SayHello failed: %v", err)
	}
	slog.Info("SayHello response", "message", reply.GetMessage())

	// --- Server Streaming RPC ---
	slog.Info("calling SayHelloServerStream (server streaming)...")
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	stream, err := client.SayHelloServerStream(ctx2, &pb.SayHelloServerStreamRequest{Name: defaultName})
	if err != nil {
		log.Fatalf("SayHelloServerStream failed: %v", err)
	}

	for {
		reply, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("stream recv failed: %v", err)
		}
		slog.Info("stream response", "message", reply.GetMessage())
	}

	slog.Info("client demo completed")
}
