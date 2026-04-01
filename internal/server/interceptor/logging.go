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
		cfg := getConfig()
		l := cfg.logger
		if l == nil {
			l = logging.Default()
		}
		start := time.Now()

		resp, err := handler(ctx, req)
		duration := time.Since(start)

		attrs := buildLogArgs(ctx, info.FullMethod, duration, err, cfg.trustProxy)

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
func StreamLogging() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		cfg := getConfig()
		l := cfg.logger
		if l == nil {
			l = logging.Default()
		}
		start := time.Now()

		err := handler(srv, ss)
		duration := time.Since(start)

		attrs := buildLogArgs(ss.Context(), info.FullMethod, duration, err, cfg.trustProxy)

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
// The trustProxy parameter must come from the caller's config snapshot so that
// a single consistent configuration generation is used for the entire request.
// When trustProxy is true, the logged peer reflects the true client IP
// extracted from proxy headers (X-Forwarded-For, X-Real-IP).
func buildLogArgs(ctx context.Context, method string, duration time.Duration, err error, trustProxy bool) []any {
	st, _ := status.FromError(err)

	args := []any{
		"method", method,
		"duration", duration,
		"code", st.Code().String(),
	}

	if key := peerKey(ctx, trustProxy); key != "unknown" {
		args = append(args, "peer", key)
	}

	if id := RequestIDFromContext(ctx); id != "" {
		args = append(args, "request_id", id)
	}

	if err != nil {
		args = append(args, "error", err.Error())
	}

	return args
}
