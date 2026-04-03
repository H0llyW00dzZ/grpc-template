// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package main demonstrates round-robin load balancing across multiple
// gRPC server instances.
//
// It starts three in-process servers on OS-assigned ports, registers
// them with a manual resolver, connects a single client configured
// with [client.WithLoadBalancing]("round_robin"), and sends a batch
// of unary RPCs and opens multiple server-streaming RPCs. Each
// response includes the server's listen address so the round-robin
// distribution is clearly visible.
//
// Run:
//
//	go run ./examples/loadbalancing
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/client"
	_ "github.com/H0llyW00dzZ/grpc-template/internal/client/balancer"
	clientinterceptor "github.com/H0llyW00dzZ/grpc-template/internal/client/interceptor"
	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/H0llyW00dzZ/grpc-template/internal/service/greeter"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
)

const (
	numServers  = 3
	numRequests = 12
	numStreams   = 6
	scheme      = "lb-demo"
	serviceName = "greeter"
)

func main() {
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))
	l := logging.Default()

	// Start multiple independent server instances.
	listeners := startServers(l)
	time.Sleep(200 * time.Millisecond)

	// Register a manual resolver with all server addresses.
	r := manual.NewBuilderWithScheme(scheme)
	resolver.Register(r)

	var addrs []resolver.Address
	for _, lis := range listeners {
		addrs = append(addrs, resolver.Address{Addr: lis.Addr().String()})
	}
	target := fmt.Sprintf("%s:///%s", scheme, serviceName)

	// Create and connect the client with round-robin balancing.
	c := client.New(target,
		client.WithInsecure(),
		client.WithLogger(l),
		client.WithUnaryInterceptors(
			clientinterceptor.Logging(),
		),
		client.WithStreamInterceptors(
			clientinterceptor.StreamLogging(),
		),
		client.WithLoadBalancing("round_robin"),
	)

	ctx := context.Background()
	if err := c.Connect(ctx); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer c.Close()

	// Push addresses into the resolver after the client is created.
	r.UpdateState(resolver.State{Addresses: addrs})

	if err := c.WaitReady(ctx); err != nil {
		log.Fatalf("wait ready: %v", err)
	}

	caller := greeter.NewCaller(c.Conn(), l)

	// Run unary and streaming demos.
	runUnaryDemo(ctx, caller, l)
	runStreamDemo(ctx, caller, l)
}
