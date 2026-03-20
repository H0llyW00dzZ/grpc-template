// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package main is the entry point for the gRPC server.
package main

import (
	"context"
	"log"

	"github.com/H0llyW00dzZ/grpc-template/internal/server"
	"github.com/H0llyW00dzZ/grpc-template/internal/service"
)

func main() {
	// Create the greeter service.
	greeterSvc := service.NewGreeterService()

	// Create and configure the gRPC server.
	srv := server.New(
		server.WithPort("50051"),
		server.WithReflection(),
		server.WithUnaryInterceptors(
			server.RecoveryInterceptor(),
			server.LoggingInterceptor(),
		),
		server.WithStreamInterceptors(
			server.StreamRecoveryInterceptor(),
			server.StreamLoggingInterceptor(),
		),
	)

	// Register services.
	srv.RegisterService(greeterSvc.Register)

	// Run the server (blocks until shutdown).
	if err := srv.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
