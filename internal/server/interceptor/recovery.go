// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Recovery returns a unary server interceptor that recovers
// from panics in RPC handlers and returns an Internal error to the client.
func Recovery() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		// Take a single config snapshot before the handler runs so
		// the deferred recover uses the same generation as any other
		// interceptor in the chain (consistent with the single-snapshot
		// contract documented in doc.go).
		cfg := getConfig()
		l := cfg.resolvedLogger()

		defer func() {
			if r := recover(); r != nil {
				l.Error("panic recovered in gRPC handler",
					"method", info.FullMethod,
					"panic", r,
					"stack", string(debug.Stack()),
				)
				err = status.Error(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// StreamRecovery returns a stream server interceptor that recovers
// from panics in streaming RPC handlers and returns an Internal error to the client.
func StreamRecovery() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		cfg := getConfig()
		l := cfg.resolvedLogger()

		defer func() {
			if r := recover(); r != nil {
				l.Error("panic recovered in gRPC stream handler",
					"method", info.FullMethod,
					"panic", r,
					"stack", string(debug.Stack()),
				)
				err = status.Error(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, ss)
	}
}
