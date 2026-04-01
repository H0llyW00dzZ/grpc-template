// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
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
		cfg := getConfig()
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

			log := cfg.logger
			if log == nil {
				log = logging.Default()
			}
			log.Warn("retrying RPC",
				"method", method,
				"attempt", attempt+1,
				"max_retries", cfg.retryMax,
				"backoff", wait,
				"error", st.Message(),
			)

			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}
		}

		return lastErr
	}
}

// maxBackoffShift is the largest safe shift for int64 exponentiation.
// Beyond this, 1<<uint(attempt) overflows and produces zero or negative values.
const maxBackoffShift = 62

// backoffDuration calculates the wait time for the given attempt using
// exponential growth with jitter. The result is uniformly distributed
// between base/2 and base × 2^attempt, capped to prevent int64 overflow.
// A non-positive base returns immediately (zero duration).
func backoffDuration(attempt int, base time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	if attempt > maxBackoffShift {
		attempt = maxBackoffShift
	}
	expBackoff := base * time.Duration(int64(1)<<uint(attempt))

	// Detect multiplication overflow: if both operands are positive
	// but the result is not, the product wrapped around.
	if expBackoff <= 0 {
		expBackoff = time.Duration(math.MaxInt64)
	}

	halfBackoff := expBackoff / 2

	if halfBackoff <= 0 {
		return expBackoff
	}

	jitter := time.Duration(rand.Int64N(int64(halfBackoff)))
	return halfBackoff + jitter
}

// isRetryable reports whether the given code is in the retryable set.
func isRetryable(code codes.Code, retryable []codes.Code) bool {
	return slices.Contains(retryable, code)
}
