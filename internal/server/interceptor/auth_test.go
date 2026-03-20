// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/server/interceptor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuth_Valid(t *testing.T) {
	authFunc := func(ctx context.Context, token string) (context.Context, error) {
		if token != "valid-token" {
			return ctx, fmt.Errorf("bad token")
		}
		return context.WithValue(ctx, "user", "alice"), nil
	}

	i := interceptor.Auth(authFunc)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Secure"}

	handler := func(ctx context.Context, req any) (any, error) {
		if ctx.Value("user") != "alice" {
			t.Error("expected enriched context with user=alice")
		}
		return "ok", nil
	}

	md := metadata.Pairs("authorization", "Bearer valid-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := i(ctx, "req", info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "ok" {
		t.Errorf("got %v, want %q", resp, "ok")
	}
}

func TestAuth_MissingToken(t *testing.T) {
	authFunc := func(ctx context.Context, token string) (context.Context, error) {
		return ctx, nil
	}

	i := interceptor.Auth(authFunc)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Secure"}

	handler := func(ctx context.Context, req any) (any, error) {
		t.Fatal("handler should not be called")
		return nil, nil
	}

	_, err := i(context.Background(), "req", info, handler)
	if err == nil {
		t.Fatal("expected Unauthenticated error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Errorf("got code %v, want Unauthenticated", st.Code())
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	authFunc := func(ctx context.Context, token string) (context.Context, error) {
		return ctx, fmt.Errorf("invalid")
	}

	i := interceptor.Auth(authFunc)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Secure"}

	handler := func(ctx context.Context, req any) (any, error) {
		t.Fatal("handler should not be called")
		return nil, nil
	}

	md := metadata.Pairs("authorization", "Bearer bad-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := i(ctx, "req", info, handler)
	if err == nil {
		t.Fatal("expected Unauthenticated error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Errorf("got code %v, want Unauthenticated", st.Code())
	}
}

func TestAuth_ExcludedMethod(t *testing.T) {
	authFunc := func(ctx context.Context, token string) (context.Context, error) {
		t.Fatal("authFunc should not be called for excluded methods")
		return ctx, nil
	}

	i := interceptor.Auth(authFunc,
		interceptor.WithExcludedMethods("/grpc.health.v1.Health/Check"),
	)
	info := &grpc.UnaryServerInfo{FullMethod: "/grpc.health.v1.Health/Check"}

	handler := func(ctx context.Context, req any) (any, error) {
		return "healthy", nil
	}

	resp, err := i(context.Background(), "req", info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "healthy" {
		t.Errorf("got %v, want %q", resp, "healthy")
	}
}

func TestAuth_BearerCaseInsensitive(t *testing.T) {
	authFunc := func(ctx context.Context, token string) (context.Context, error) {
		if token != "my-token" {
			return ctx, fmt.Errorf("unexpected token: %s", token)
		}
		return ctx, nil
	}

	i := interceptor.Auth(authFunc)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Secure"}

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	md := metadata.Pairs("authorization", "bearer my-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := i(ctx, "req", info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuth_EmptyBearer(t *testing.T) {
	authFunc := func(ctx context.Context, token string) (context.Context, error) {
		return ctx, nil
	}

	i := interceptor.Auth(authFunc)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Secure"}

	handler := func(ctx context.Context, req any) (any, error) {
		t.Fatal("handler should not be called")
		return nil, nil
	}

	md := metadata.Pairs("authorization", "Bearer ")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := i(ctx, "req", info, handler)
	if err == nil {
		t.Fatal("expected Unauthenticated error for empty bearer, got nil")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Errorf("got code %v, want Unauthenticated", st.Code())
	}
}

func TestStreamAuth_Valid(t *testing.T) {
	authFunc := func(ctx context.Context, token string) (context.Context, error) {
		if token != "stream-token" {
			return ctx, fmt.Errorf("bad token")
		}
		return context.WithValue(ctx, "user", "bob"), nil
	}

	i := interceptor.StreamAuth(authFunc)
	info := &grpc.StreamServerInfo{FullMethod: "/test.v1.Svc/SecureStream"}

	handler := func(srv any, stream grpc.ServerStream) error {
		if stream.Context().Value("user") != "bob" {
			t.Error("expected enriched context with user=bob")
		}
		return nil
	}

	md := metadata.Pairs("authorization", "Bearer stream-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ss := &fakeServerStream{ctx: ctx}

	err := i(nil, ss, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStreamAuth_ExcludedMethod(t *testing.T) {
	authFunc := func(ctx context.Context, token string) (context.Context, error) {
		t.Fatal("authFunc should not be called for excluded methods")
		return ctx, nil
	}

	i := interceptor.StreamAuth(authFunc,
		interceptor.WithExcludedMethods("/grpc.reflection.v1.ServerReflection/ServerReflectionInfo"),
	)
	info := &grpc.StreamServerInfo{FullMethod: "/grpc.reflection.v1.ServerReflection/ServerReflectionInfo"}
	ss := &fakeServerStream{ctx: context.Background()}

	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}

	err := i(nil, ss, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
