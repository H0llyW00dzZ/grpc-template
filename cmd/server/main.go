// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package main is the entry point for the gRPC server.
package main

import (
	"context"
	"log"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/H0llyW00dzZ/grpc-template/internal/server"
	"github.com/H0llyW00dzZ/grpc-template/internal/server/interceptor"
	"github.com/H0llyW00dzZ/grpc-template/internal/service/greeter"
)

func main() {
	// Initialize logger
	l := logging.Default()

	// Create and configure the gRPC server.
	srv := server.New(
		server.WithPort("50051"),
		server.WithReflection(),
		server.WithLogger(l),
		server.WithUnaryInterceptors(
			interceptor.Recovery(),
			interceptor.Logging(),
		),
		server.WithStreamInterceptors(
			interceptor.StreamRecovery(),
			interceptor.StreamLogging(),
		),
	)

	// Create the greeter service utilizing the server's integrated logger.
	greeterSvc := greeter.NewService(srv.Logger())

	// Register services.
	srv.RegisterService(greeterSvc.Register)

	// Run the server (blocks until shutdown).
	if err := srv.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
