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
	"os"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/client"
	clientinterceptor "github.com/H0llyW00dzZ/grpc-template/internal/client/interceptor"
	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/H0llyW00dzZ/grpc-template/internal/service/greeter"
)

const (
	defaultAddr = "dns:///localhost:50051"
	defaultName = "Gopher"
)

func main() {
	// Enable debug logging (shows Debug level + reflection calls)
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(h))
	// Initialize logger.
	l := logging.Default()

	// Create and configure the gRPC client.
	c := client.New(defaultAddr,
		client.WithInsecure(),
		client.WithLogger(l),
		client.WithDefaultTimeout(5*time.Second),
		client.WithRetry(3, time.Second),
		client.WithUnaryInterceptors(
			clientinterceptor.Logging(),
			clientinterceptor.Timeout(),
			clientinterceptor.Retry(),
		),
		client.WithStreamInterceptors(
			clientinterceptor.StreamLogging(),
		),
		client.WithLoadBalancing("round_robin"),
	)

	// Connect to the server.
	ctx := context.Background()
	if err := c.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// List available services via runtime reflection.
	services, err := c.ListServices(ctx)
	if err != nil {
		l.Error("ListServices: %v", err)
	} else {
		for _, svc := range services {
			l.Info("available service", "name", svc)
		}
	}

	// Create the greeter caller using the client's connection and logger.
	caller := greeter.NewCaller(c.Conn(), c.Logger())

	// --- Unary RPC ---
	reply, err := caller.SayHello(ctx, defaultName)
	if err != nil {
		log.Fatalf("SayHello failed: %v", err)
	}
	l.Info("SayHello response", "message", reply.GetMessage())

	// --- Server Streaming RPC ---
	stream, err := caller.SayHelloServerStream(ctx, defaultName)
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
		l.Info("stream response", "message", reply.GetMessage())
	}

	l.Info("client demo completed")
}
