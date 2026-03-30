// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"
	"math/rand/v2"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Retry returns a unary client interceptor that retries failed RPCs
// when the returned status code is in the configured retryable set.
//
// It uses exponential backoff with jitter: each attempt waits a
// random duration between backoff/2 and backoff × 2^attempt, where
// backoff is the base duration configured via [WithRetry].
//
// If no retry configuration has been set (maxRetries ≤ 0), the
// interceptor is a no-op and the RPC is invoked exactly once.
//
//	interceptor.Configure(
//	    interceptor.WithRetry(3, time.Second),
//	)
func Retry() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		cfg := defaultConfig
		if cfg.retryMax <= 0 {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		var lastErr error
		for attempt := range cfg.retryMax + 1 {
			lastErr = invoker(ctx, method, req, reply, cc, opts...)
			if lastErr == nil {
				return nil
			}

			st, ok := status.FromError(lastErr)
			if !ok || !isRetryable(st.Code(), cfg.retryCodes) {
				return lastErr
			}

			// Don't sleep after the last attempt.
			if attempt >= cfg.retryMax {
				break
			}

			wait := backoffDuration(attempt, cfg.retryBackoff)

			logger().Warn("retrying RPC",
				"method", method,
				"attempt", attempt+1,
				"max_retries", cfg.retryMax,
				"backoff", wait,
				"error", st.Message(),
			)

			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		return lastErr
	}
}

// backoffDuration calculates the wait time for the given attempt using
// exponential growth with jitter. The result is uniformly distributed
// between base/2 and base × 2^attempt.
func backoffDuration(attempt int, base time.Duration) time.Duration {
	expBackoff := base * time.Duration(int64(1)<<uint(attempt))
	halfBackoff := expBackoff / 2

	if halfBackoff <= 0 {
		return expBackoff
	}

	jitter := time.Duration(rand.Int64N(int64(halfBackoff)))
	return halfBackoff + jitter
}

// isRetryable reports whether the given code is in the retryable set.
func isRetryable(code codes.Code, retryable []codes.Code) bool {
	for _, c := range retryable {
		if code == c {
			return true
		}
	}
	return false
}
