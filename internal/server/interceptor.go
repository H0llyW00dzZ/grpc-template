// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package server

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor returns a unary server interceptor that logs
// the method name, duration, and any error for each RPC call.
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)
		duration := time.Since(start)

		attrs := []slog.Attr{
			slog.String("method", info.FullMethod),
			slog.Duration("duration", duration),
		}

		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
			slog.LogAttrs(ctx, slog.LevelError, "rpc failed", attrs...)
		} else {
			slog.LogAttrs(ctx, slog.LevelInfo, "rpc completed", attrs...)
		}

		return resp, err
	}
}

// RecoveryInterceptor returns a unary server interceptor that recovers
// from panics in RPC handlers and returns an Internal error to the client.
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered in gRPC handler",
					"method", info.FullMethod,
					"panic", r,
					"stack", string(debug.Stack()),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}
