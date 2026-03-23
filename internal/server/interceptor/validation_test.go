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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type validMessage struct{}

func (v *validMessage) Validate() error { return nil }

type invalidMessage struct{}

func (v *invalidMessage) Validate() error { return fmt.Errorf("field 'name' is required") }

func TestValidation_Valid(t *testing.T) {
	i := interceptor.Validation()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Create"}

	handler := func(ctx context.Context, req any) (any, error) {
		return "created", nil
	}

	resp, err := i(context.Background(), &validMessage{}, info, handler)
	require.NoError(t, err)
	assert.Equal(t, "created", resp)
}

func TestValidation_Invalid(t *testing.T) {
	i := interceptor.Validation()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Create"}

	handler := func(ctx context.Context, req any) (any, error) {
		t.Fatal("handler should not be called for invalid request")
		return nil, nil
	}

	_, err := i(context.Background(), &invalidMessage{}, info, handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestValidation_NoValidateMethod(t *testing.T) {
	i := interceptor.Validation()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Create"}

	handler := func(ctx context.Context, req any) (any, error) {
		return "passed", nil
	}

	resp, err := i(context.Background(), "plain-request", info, handler)
	require.NoError(t, err)
	assert.Equal(t, "passed", resp)
}
