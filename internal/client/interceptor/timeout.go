// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"

	"google.golang.org/grpc"
)

// Timeout returns a unary client interceptor that enforces a default
// deadline on RPCs when no deadline is already set on the context.
//
// The timeout value is configured via [WithDefaultTimeout]:
//
//	interceptor.Configure(
//	    interceptor.WithDefaultTimeout(5 * time.Second),
//	)
//
// If no timeout has been configured, the interceptor is a no-op
// and the original context is passed through unchanged.
func Timeout() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		timeout := getConfig().defaultTimeout
		if timeout > 0 {
			if _, ok := ctx.Deadline(); !ok {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
