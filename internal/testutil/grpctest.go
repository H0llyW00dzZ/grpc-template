// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package testutil provides shared test helpers for gRPC services.
package testutil

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// NewBufListener creates an in-memory listener for gRPC testing.
// No real TCP port is opened.
func NewBufListener() *bufconn.Listener {
	return bufconn.Listen(bufSize)
}

// DialBufNet creates a gRPC client connection to an in-memory bufconn listener.
// The caller is responsible for closing the returned connection.
func DialBufNet(ctx context.Context, lis *bufconn.Listener) (*grpc.ClientConn, error) {
	return grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

// BufDialer returns a context dialer function for the given bufconn listener.
// Use this with [grpc.WithContextDialer] or [client.WithDialOptions] when
// creating test clients.
//
//	c := client.New("passthrough:///bufconn",
//	    client.WithDialOptions(
//	        grpc.WithContextDialer(testutil.BufDialer(lis)),
//	        grpc.WithTransportCredentials(insecure.NewCredentials()),
//	    ),
//	)
func BufDialer(lis *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}
}
