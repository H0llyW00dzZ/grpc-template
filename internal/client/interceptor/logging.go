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

// Logging returns a unary client interceptor that logs each RPC call
// with its method name, duration, and resulting gRPC status code.
//
// It uses the logger configured via [Configure] or [logging.Default].
// A single configuration snapshot is taken at the start of each RPC
// to ensure consistent logger usage throughout the request.
func Logging() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		cfg := getConfig()
		l := logging.Resolve(cfg.logger)
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		duration := time.Since(start)

		st, _ := status.FromError(err)

		if err != nil {
			l.Error("RPC failed",
				"method", method,
				"code", st.Code().String(),
				"duration", duration,
				"error", st.Message(),
			)
		} else {
			l.Info("RPC completed",
				"method", method,
				"code", st.Code().String(),
				"duration", duration,
			)
		}

		return err
	}
}

// StreamLogging returns a streaming client interceptor that logs when
// a stream is opened and the resulting gRPC status code.
//
// It uses the logger configured via [Configure] or [logging.Default].
// A single configuration snapshot is taken at the start of each stream
// to ensure consistent logger usage throughout the request.
func StreamLogging() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		cfg := getConfig()
		l := logging.Resolve(cfg.logger)
		l.Info("stream opening", "method", method)

		start := time.Now()
		cs, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			st, _ := status.FromError(err)
			l.Error("stream failed to open",
				"method", method,
				"code", st.Code().String(),
				"duration", time.Since(start),
				"error", st.Message(),
			)
			return nil, err
		}

		return cs, nil
	}
}
