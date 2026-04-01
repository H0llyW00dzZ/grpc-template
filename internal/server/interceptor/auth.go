// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Auth returns a unary server interceptor that extracts a bearer
// token from the "authorization" metadata and validates it using the
// [AuthFunc] configured via [Configure] with [WithAuthFunc].
//
// Use [WithExcludedMethods] to skip authentication for specific
// methods (e.g., health checks, reflection).
//
//	interceptor.Configure(
//	    interceptor.WithAuthFunc(func(ctx context.Context, token string) (context.Context, error) {
//	        claims, err := validateJWT(token)
//	        if err != nil {
//	            return ctx, err
//	        }
//	        return context.WithValue(ctx, claimsKey{}, claims), nil
//	    }),
//	    interceptor.WithExcludedMethods("/grpc.health.v1.Health/Check"),
//	)
func Auth() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		cfg := getConfig()
		if _, excluded := cfg.excludedMethods[info.FullMethod]; excluded {
			return handler(ctx, req)
		}

		fn := cfg.authFunc
		if fn == nil {
			return handler(ctx, req)
		}

		ctx, err := authenticateContext(ctx, fn)
		if err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

// StreamAuth returns a stream server interceptor that extracts a
// bearer token from the "authorization" metadata and validates it using the
// [AuthFunc] configured via [Configure] with [WithAuthFunc].
func StreamAuth() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		cfg := getConfig()
		if _, excluded := cfg.excludedMethods[info.FullMethod]; excluded {
			return handler(srv, ss)
		}

		fn := cfg.authFunc
		if fn == nil {
			return handler(srv, ss)
		}

		ctx, err := authenticateContext(ss.Context(), fn)
		if err != nil {
			return err
		}

		return handler(srv, &wrappedServerStream{ServerStream: ss, ctx: ctx})
	}
}

// authenticateContext extracts the bearer token from metadata and calls the
// AuthFunc. Returns an enriched context or a gRPC Unauthenticated error.
//
// The real error from the AuthFunc is logged internally for debugging;
// clients always receive a generic "authentication failed" message to
// avoid leaking internal details (OWASP / CWE-209).
func authenticateContext(ctx context.Context, fn AuthFunc) (context.Context, error) {
	token, err := extractBearerToken(ctx)
	if err != nil {
		return ctx, err
	}

	ctx, err = fn(ctx, token)
	if err != nil {
		// Return a generic message to the client — never expose internal details
		// (OWASP / CWE-209). The Logging interceptor already captures every
		// failed RPC, so no additional logging is needed here.
		return ctx, status.Error(codes.Unauthenticated, "authentication failed")
	}

	return ctx, nil
}

// extractBearerToken extracts the bearer token from the "authorization"
// metadata key. It expects the format "Bearer <token>" or "bearer <token>".
func extractBearerToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	vals := md.Get("authorization")
	if len(vals) == 0 || vals[0] == "" {
		return "", status.Error(codes.Unauthenticated, "missing authorization token")
	}

	// Support "Bearer <token>" format.
	token := vals[0]
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = token[7:]
	}

	if token == "" {
		return "", status.Error(codes.Unauthenticated, "empty bearer token")
	}

	return token, nil
}
