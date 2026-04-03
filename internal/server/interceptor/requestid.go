// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"context"
	"crypto/rand"
	"fmt"
	"regexp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// uuidPattern matches a canonical UUID (8-4-4-4-12 hex).
var uuidPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

// requestIDKey is the private context key for the request ID.
type requestIDKey struct{}

// RequestIDFromContext extracts the request ID from the context.
// Returns an empty string if no request ID is present.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// RequestID returns a unary server interceptor that extracts or
// generates a unique request ID for each RPC. The ID is read from the
// "x-request-id" metadata key. If absent, a new UUID is generated.
//
// The request ID is stored in the context (retrieve via [RequestIDFromContext])
// and sent back in the response header metadata.
func RequestID() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		ctx = ensureRequestID(ctx)

		// Set the request ID in response headers.
		id := RequestIDFromContext(ctx)
		_ = grpc.SetHeader(ctx, metadata.Pairs("x-request-id", id))

		return handler(ctx, req)
	}
}

// StreamRequestID returns a stream server interceptor that extracts
// or generates a unique request ID for each streaming RPC.
func StreamRequestID() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ensureRequestID(ss.Context())

		// Set the request ID in response headers.
		id := RequestIDFromContext(ctx)
		_ = grpc.SetHeader(ctx, metadata.Pairs("x-request-id", id))

		return handler(srv, &wrappedServerStream{ServerStream: ss, ctx: ctx})
	}
}

// ensureRequestID extracts the request ID from incoming metadata and validates
// it against a strict UUID format. If the value is missing or does not match,
// a new server-generated UUID is used instead. This prevents spoofing, log
// injection, and oversized payloads from untrusted clients.
func ensureRequestID(ctx context.Context) context.Context {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-request-id"); len(vals) > 0 && uuidPattern.MatchString(vals[0]) {
			return context.WithValue(ctx, requestIDKey{}, vals[0])
		}
	}

	return context.WithValue(ctx, requestIDKey{}, generateUUID())
}

// generateUUID produces a version 4 (random) UUID string per [RFC 9562]
// using [crypto/rand] as the entropy source.
//
// [crypto/rand] uses the platform's cryptographic random source
// (getrandom(2) on Linux, CryptGenRandom on Windows, /dev/urandom as
// fallback) which is always available after boot, so the error from
// [rand.Read] is safe to discard.
//
// [RFC 9562]: https://datatracker.ietf.org/doc/html/rfc9562#section-5.4
func generateUUID() string {
	var uuid [16]byte
	_, _ = rand.Read(uuid[:])
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
