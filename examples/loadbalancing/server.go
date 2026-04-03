// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package main

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/H0llyW00dzZ/grpc-template/internal/server"
	"github.com/H0llyW00dzZ/grpc-template/internal/server/interceptor"
	"github.com/H0llyW00dzZ/grpc-template/internal/service/greeter"
)

// startServers creates numServers gRPC server instances on OS-assigned
// ports and runs them in background goroutines. It returns the listeners
// so the caller can extract their addresses for the resolver.
func startServers(l logging.Handler) []net.Listener {
	var listeners []net.Listener
	var wg sync.WaitGroup

	for i := range numServers {
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatalf("listen: %v", err)
		}
		listeners = append(listeners, lis)
		addr := lis.Addr().String()

		srv := server.New(
			server.WithListener(lis),
			server.WithReflection(),
			server.WithLogger(l),
			server.WithUnaryInterceptors(
				interceptor.RequestID(),
				serverTag(addr),
				interceptor.Logging(),
			),
			server.WithStreamInterceptors(
				interceptor.StreamRequestID(),
				streamServerTag(addr),
				interceptor.StreamLogging(),
			),
		)

		svc := greeter.NewService(l)
		srv.RegisterService(svc.Register)

		wg.Add(1)
		go func(idx int, s *server.Server) {
			defer wg.Done()
			if err := s.Run(context.Background()); err != nil {
				l.Error("server stopped", "index", idx, "error", err)
			}
		}(i, srv)

		l.Info("server started", "index", i, "address", addr)
	}

	return listeners
}
