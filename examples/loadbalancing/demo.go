// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package main

import (
	"context"
	"fmt"
	"io"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/H0llyW00dzZ/grpc-template/internal/service/greeter"
	pb "github.com/H0llyW00dzZ/grpc-template/pkg/gen/helloworld/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// runUnaryDemo sends numRequests unary RPCs and logs which server
// handled each one, then prints a distribution summary.
func runUnaryDemo(ctx context.Context, caller *greeter.Caller, l logging.Handler) {
	l.Info("starting unary load balancing demo",
		"servers", numServers,
		"requests", numRequests,
		"policy", "round_robin",
	)

	hits := make(map[string]int)

	for i := range numRequests {
		var header metadata.MD
		reply, err := caller.SayHello(
			ctx,
			fmt.Sprintf("Request-%d", i+1),
			grpc.Header(&header),
		)
		if err != nil {
			l.Error("SayHello failed", "request", i+1, "error", err)
			continue
		}

		serverAddr := addrFromHeader(header)
		hits[serverAddr]++

		l.Info("request routed",
			"request", i+1,
			"server", serverAddr,
			"reply", reply.GetMessage(),
		)
	}

	logSummary(l, "unary", hits)
}

// runStreamDemo opens numStreams server-streaming RPCs and logs which
// server handled each stream, then prints a distribution summary.
func runStreamDemo(ctx context.Context, caller *greeter.Caller, l logging.Handler) {
	l.Info("starting streaming load balancing demo",
		"streams", numStreams,
		"policy", "round_robin",
	)

	hits := make(map[string]int)

	for i := range numStreams {
		var header metadata.MD
		stream, err := caller.SayHelloServerStream(
			ctx,
			fmt.Sprintf("Stream-%d", i+1),
			grpc.Header(&header),
		)
		if err != nil {
			l.Error("SayHelloServerStream failed", "stream", i+1, "error", err)
			continue
		}

		msgCount := drainStream(stream)
		serverAddr := addrFromHeader(header)
		hits[serverAddr]++

		l.Info("stream routed",
			"stream", i+1,
			"server", serverAddr,
			"messages", msgCount,
		)
	}

	logSummary(l, "stream", hits)
}

// addrFromHeader extracts the x-server-addr value from response metadata.
func addrFromHeader(header metadata.MD) string {
	if vals := header.Get("x-server-addr"); len(vals) > 0 {
		return vals[0]
	}
	return "unknown"
}

// drainStream consumes all messages from a server stream and returns the count.
func drainStream(stream pb.GreeterService_SayHelloServerStreamClient) int {
	count := 0
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		count++
	}
	return count
}

// logSummary prints the per-server distribution for a given RPC type.
func logSummary(l logging.Handler, rpcType string, hits map[string]int) {
	l.Info(rpcType + " distribution summary")
	for addr, count := range hits {
		l.Info("server handled",
			"type", rpcType,
			"address", addr,
			"count", count,
		)
	}
}
