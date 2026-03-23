// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// Logging returns a unary server interceptor that logs
// the method name, duration, gRPC status code, peer address,
// and any error for each RPC call.
func Logging(l logging.Handler) grpc.UnaryServerInterceptor {
	if l == nil {
		l = logging.Default()
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)
		duration := time.Since(start)

		attrs := buildLogArgs(ctx, info.FullMethod, duration, err)

		if err != nil {
			l.Error("rpc failed", attrs...)
		} else {
			l.Info("rpc completed", attrs...)
		}

		return resp, err
	}
}

// StreamLogging returns a stream server interceptor that logs
// the method name, duration, gRPC status code, peer address,
// and any error for each streaming RPC call.
func StreamLogging(l logging.Handler) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		err := handler(srv, ss)
		duration := time.Since(start)

		attrs := buildLogArgs(ss.Context(), info.FullMethod, duration, err)

		if err != nil {
			l.Error("stream rpc failed", attrs...)
		} else {
			l.Info("stream rpc completed", attrs...)
		}

		return err
	}
}

// buildLogArgs creates the common key-value pairs for both unary and stream
// logging interceptors, including method, duration, gRPC status code, and peer address.
func buildLogArgs(ctx context.Context, method string, duration time.Duration, err error) []any {
	st, _ := status.FromError(err)

	args := []any{
		"method", method,
		"duration", duration,
		"code", st.Code().String(),
	}

	if p, ok := peer.FromContext(ctx); ok {
		args = append(args, "peer", p.Addr.String())
	}

	if id := RequestIDFromContext(ctx); id != "" {
		args = append(args, "request_id", id)
	}

	if err != nil {
		args = append(args, "error", err.Error())
	}

	return args
}
