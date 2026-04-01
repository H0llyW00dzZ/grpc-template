// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Auth returns a unary client interceptor that injects a bearer token
// into the outgoing gRPC metadata using the configured [TokenSource].
//
// The token source is set via [WithTokenSource]:
//
//	interceptor.Configure(
//	    interceptor.WithTokenSource(interceptor.StaticToken("my-token")),
//	)
//
// If no token source has been configured, the interceptor is a no-op.
func Auth() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		cfg := getConfig()
		ctx, err := injectToken(ctx, cfg.tokenSource)
		if err != nil {
			return err
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamAuth returns a streaming client interceptor that injects a bearer
// token into the outgoing gRPC metadata using the configured [TokenSource].
//
// If no token source has been configured, the interceptor is a no-op.
func StreamAuth() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		cfg := getConfig()
		ctx, err := injectToken(ctx, cfg.tokenSource)
		if err != nil {
			return nil, err
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// injectToken calls the given TokenSource and injects the bearer token
// into outgoing metadata. The caller passes the token source from its
// own config snapshot so that a single snapshot is used per request.
func injectToken(ctx context.Context, src TokenSource) (context.Context, error) {
	if src == nil {
		return ctx, nil
	}

	ctx, err := src(ctx)
	if err != nil {
		return ctx, status.Errorf(codes.Unauthenticated, "client interceptor: token source failed: %v", err)
	}

	token, ok := ctx.Value(tokenKey{}).(string)
	if !ok || token == "" {
		return ctx, status.Error(codes.Unauthenticated, "client interceptor: no token in context")
	}

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}
	md.Set("authorization", "Bearer "+token)

	return metadata.NewOutgoingContext(ctx, md), nil
}
