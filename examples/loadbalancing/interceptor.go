// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package main

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// serverTag returns a unary interceptor that injects the server's
// listen address into the response header so the client can see
// which backend handled each request.
func serverTag(addr string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		_ = grpc.SetHeader(ctx, metadata.Pairs("x-server-addr", addr))
		return handler(ctx, req)
	}
}

// streamServerTag returns a stream interceptor that injects the server's
// listen address into the response header.
func streamServerTag(addr string) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		_ = grpc.SetHeader(ss.Context(), metadata.Pairs("x-server-addr", addr))
		return handler(srv, ss)
	}
}
