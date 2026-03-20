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

// AuthFunc is a user-provided function that validates a token string
// and returns an enriched context (e.g., with user claims) or an error.
// The interceptor extracts the bearer token from the "authorization"
// metadata key and passes it to this function.
type AuthFunc func(ctx context.Context, token string) (context.Context, error)

// AuthOption configures the auth interceptor.
type AuthOption func(*authConfig)

type authConfig struct {
	excludedMethods map[string]struct{}
}

// WithExcludedMethods returns an AuthOption that skips authentication
// for the given fully-qualified gRPC method names.
//
//	interceptor.Auth(myAuthFunc,
//	    interceptor.WithExcludedMethods(
//	        "/grpc.health.v1.Health/Check",
//	        "/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
//	    ),
//	)
func WithExcludedMethods(methods ...string) AuthOption {
	return func(c *authConfig) {
		for _, m := range methods {
			c.excludedMethods[m] = struct{}{}
		}
	}
}

// Auth returns a unary server interceptor that extracts a bearer
// token from the "authorization" metadata and validates it using the provided
// [AuthFunc]. Use [WithExcludedMethods] to skip authentication for specific
// methods (e.g., health checks, reflection).
//
//	interceptor.Auth(func(ctx context.Context, token string) (context.Context, error) {
//	    claims, err := validateJWT(token)
//	    if err != nil {
//	        return ctx, err
//	    }
//	    return context.WithValue(ctx, claimsKey{}, claims), nil
//	})
func Auth(fn AuthFunc, opts ...AuthOption) grpc.UnaryServerInterceptor {
	cfg := newAuthConfig(opts)

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if _, excluded := cfg.excludedMethods[info.FullMethod]; excluded {
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
// provided [AuthFunc].
func StreamAuth(fn AuthFunc, opts ...AuthOption) grpc.StreamServerInterceptor {
	cfg := newAuthConfig(opts)

	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if _, excluded := cfg.excludedMethods[info.FullMethod]; excluded {
			return handler(srv, ss)
		}

		ctx, err := authenticateContext(ss.Context(), fn)
		if err != nil {
			return err
		}

		return handler(srv, &wrappedServerStream{ServerStream: ss, ctx: ctx})
	}
}

func newAuthConfig(opts []AuthOption) *authConfig {
	cfg := &authConfig{excludedMethods: make(map[string]struct{})}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// authenticateContext extracts the bearer token from metadata and calls the
// AuthFunc. Returns an enriched context or a gRPC Unauthenticated error.
func authenticateContext(ctx context.Context, fn AuthFunc) (context.Context, error) {
	token, err := extractBearerToken(ctx)
	if err != nil {
		return ctx, err
	}

	ctx, err = fn(ctx, token)
	if err != nil {
		return ctx, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
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
