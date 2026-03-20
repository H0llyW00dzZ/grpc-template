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
	"google.golang.org/grpc/status"
)

// validMessage implements interceptor.Validator and returns nil from Validate.
type validMessage struct{}

func (v *validMessage) Validate() error { return nil }

// invalidMessage implements interceptor.Validator and returns an error.
type invalidMessage struct{}

func (v *invalidMessage) Validate() error { return fmt.Errorf("field 'name' is required") }

func TestValidation_Valid(t *testing.T) {
	i := interceptor.Validation()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Create"}

	handler := func(ctx context.Context, req any) (any, error) {
		return "created", nil
	}

	resp, err := i(context.Background(), &validMessage{}, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "created" {
		t.Errorf("got %v, want %q", resp, "created")
	}
}

func TestValidation_Invalid(t *testing.T) {
	i := interceptor.Validation()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Create"}

	handler := func(ctx context.Context, req any) (any, error) {
		t.Fatal("handler should not be called for invalid request")
		return nil, nil
	}

	_, err := i(context.Background(), &invalidMessage{}, info, handler)
	if err == nil {
		t.Fatal("expected InvalidArgument error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("got code %v, want InvalidArgument", st.Code())
	}
}

func TestValidation_NoValidateMethod(t *testing.T) {
	i := interceptor.Validation()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Create"}

	handler := func(ctx context.Context, req any) (any, error) {
		return "passed", nil
	}

	resp, err := i(context.Background(), "plain-request", info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "passed" {
		t.Errorf("got %v, want %q", resp, "passed")
	}
}
