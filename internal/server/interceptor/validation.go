// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Validator is an interface for protobuf messages that support validation.
// This is compatible with protoc-gen-validate and buf validate generated code.
type Validator interface {
	Validate() error
}

// Validation returns a unary server interceptor that validates
// incoming requests. If the request implements the [Validator] interface
// (i.e., has a Validate() error method), the interceptor calls it and
// returns codes.InvalidArgument if validation fails.
//
// This is unary-only because stream messages arrive incrementally and
// should be validated in the handler.
func Validation() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if v, ok := req.(Validator); ok {
			if err := v.Validate(); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "request validation failed: %v", err)
			}
		}

		return handler(ctx, req)
	}
}
