// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// Logging returns a unary server interceptor that logs
// the method name, duration, gRPC status code, peer address,
// and any error for each RPC call.
func Logging() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)
		duration := time.Since(start)

		attrs := buildLogAttrs(ctx, info.FullMethod, duration, err)

		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "rpc failed", attrs...)
		} else {
			slog.LogAttrs(ctx, slog.LevelInfo, "rpc completed", attrs...)
		}

		return resp, err
	}
}

// StreamLogging returns a stream server interceptor that logs
// the method name, duration, gRPC status code, peer address,
// and any error for each streaming RPC call.
func StreamLogging() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		err := handler(srv, ss)
		duration := time.Since(start)

		attrs := buildLogAttrs(ss.Context(), info.FullMethod, duration, err)

		if err != nil {
			slog.LogAttrs(ss.Context(), slog.LevelError, "stream rpc failed", attrs...)
		} else {
			slog.LogAttrs(ss.Context(), slog.LevelInfo, "stream rpc completed", attrs...)
		}

		return err
	}
}

// buildLogAttrs creates the common slog attributes for both unary and stream
// logging interceptors, including method, duration, gRPC status code, and peer address.
func buildLogAttrs(ctx context.Context, method string, duration time.Duration, err error) []slog.Attr {
	st, _ := status.FromError(err)

	attrs := []slog.Attr{
		slog.String("method", method),
		slog.Duration("duration", duration),
		slog.String("code", st.Code().String()),
	}

	if p, ok := peer.FromContext(ctx); ok {
		attrs = append(attrs, slog.String("peer", p.Addr.String()))
	}

	if id := RequestIDFromContext(ctx); id != "" {
		attrs = append(attrs, slog.String("request_id", id))
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	return attrs
}
