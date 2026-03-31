// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor_test

import (
	"context"
	"testing"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/client/interceptor"
	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestAuth_WithToken(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(
		interceptor.WithTokenSource(interceptor.StaticToken("test-token")),
	)
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Auth()

	var capturedCtx context.Context
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		capturedCtx = ctx
		return nil
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	require.NoError(t, err)

	md, ok := metadata.FromOutgoingContext(capturedCtx)
	require.True(t, ok)
	assert.Equal(t, []string{"Bearer test-token"}, md.Get("authorization"))
}

func TestAuth_NoTokenSource(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Auth()

	called := false
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		called = true
		// No authorization metadata should be present.
		md, _ := metadata.FromOutgoingContext(ctx)
		assert.Empty(t, md.Get("authorization"))
		return nil
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestAuth_TokenSourceError(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(
		interceptor.WithTokenSource(func(ctx context.Context) (context.Context, error) {
			return ctx, assert.AnError
		}),
	)
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Auth()

	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		t.Fatal("invoker should not be called when token source errors")
		return nil
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token source failed")
}

func TestAuth_ExistingMetadata(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(
		interceptor.WithTokenSource(interceptor.StaticToken("my-token")),
	)
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Auth()

	// Set up a context that already has outgoing metadata.
	existingMD := metadata.Pairs("x-custom", "value")
	ctx := metadata.NewOutgoingContext(context.Background(), existingMD)

	var capturedCtx context.Context
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		capturedCtx = ctx
		return nil
	}

	err := i(ctx, "/test/Method", nil, nil, nil, invoker)
	require.NoError(t, err)

	md, ok := metadata.FromOutgoingContext(capturedCtx)
	require.True(t, ok)
	assert.Equal(t, []string{"Bearer my-token"}, md.Get("authorization"))
	assert.Equal(t, []string{"value"}, md.Get("x-custom"))
}

type mockClientStream struct {
	grpc.ClientStream
}

// failingTokenSource implements oauth2.TokenSource and always returns an error.
type failingTokenSource struct{}

func (f *failingTokenSource) Token() (*oauth2.Token, error) {
	return nil, assert.AnError
}

func TestStreamAuth_WithToken(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(
		interceptor.WithTokenSource(interceptor.StaticToken("stream-token")),
	)
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.StreamAuth()

	var capturedCtx context.Context
	streamer := func(ctx context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		capturedCtx = ctx
		return &mockClientStream{}, nil
	}

	cs, err := i(context.Background(), &grpc.StreamDesc{}, nil, "/test/Stream", streamer)
	require.NoError(t, err)
	require.NotNil(t, cs)

	md, ok := metadata.FromOutgoingContext(capturedCtx)
	require.True(t, ok)
	assert.Equal(t, []string{"Bearer stream-token"}, md.Get("authorization"))
}

func TestStreamAuth_NoTokenSource(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.StreamAuth()

	called := false
	streamer := func(_ context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		called = true
		return &mockClientStream{}, nil
	}

	cs, err := i(context.Background(), &grpc.StreamDesc{}, nil, "/test/Stream", streamer)
	require.NoError(t, err)
	require.NotNil(t, cs)
	assert.True(t, called)
}

func TestStreamAuth_TokenSourceError(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(
		interceptor.WithTokenSource(func(ctx context.Context) (context.Context, error) {
			return ctx, assert.AnError
		}),
	)
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.StreamAuth()

	streamer := func(_ context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		t.Fatal("streamer should not be called when token source errors")
		return nil, nil
	}

	cs, err := i(context.Background(), &grpc.StreamDesc{}, nil, "/test/Stream", streamer)
	require.Error(t, err)
	assert.Nil(t, cs)
}

func TestLogger_Fallback(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	l := interceptor.TestableLogger()
	require.NotNil(t, l)
	// Should fall back to logging.Default()
	assert.Equal(t, logging.Default(), l)
}

func TestLogger_Configured(t *testing.T) {
	interceptor.ResetConfig()
	l := logging.Default()
	interceptor.Configure(interceptor.WithLogger(l))
	t.Cleanup(interceptor.ResetConfig)

	got := interceptor.TestableLogger()
	assert.Equal(t, l, got)
}

func TestWithTokenSource(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	src := interceptor.StaticToken("configured")
	interceptor.Configure(interceptor.WithTokenSource(src))

	// Verify by calling Auth interceptor which uses the configured source.
	i := interceptor.Auth()

	var capturedCtx context.Context
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		capturedCtx = ctx
		return nil
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	require.NoError(t, err)

	md, _ := metadata.FromOutgoingContext(capturedCtx)
	assert.Equal(t, []string{"Bearer configured"}, md.Get("authorization"))
}

func TestConfigure_AllOptions(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	interceptor.Configure(
		interceptor.WithLogger(logging.Default()),
		interceptor.WithDefaultTimeout(5*time.Second),
		interceptor.WithRetry(3, time.Second),
		interceptor.WithRetryCodes(),
		interceptor.WithTokenSource(interceptor.StaticToken("tok")),
	)
}

func TestAuth_NoTokenInContext(t *testing.T) {
	interceptor.ResetConfig()
	interceptor.Configure(
		interceptor.WithTokenSource(func(ctx context.Context) (context.Context, error) {
			return ctx, nil // valid call but no token in context
		}),
	)
	t.Cleanup(interceptor.ResetConfig)

	i := interceptor.Auth()

	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		t.Fatal("invoker should not be called")
		return nil
	}

	err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no token in context")
}

func TestOAuth2TokenSource(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		interceptor.ResetConfig()
		t.Cleanup(interceptor.ResetConfig)

		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "oauth-test-token"})
		interceptor.Configure(interceptor.WithTokenSource(interceptor.OAuth2TokenSource(ts)))

		i := interceptor.Auth()

		var captured string
		invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			md, _ := metadata.FromOutgoingContext(ctx)
			if vals := md.Get("authorization"); len(vals) > 0 {
				captured = vals[0]
			}
			return nil
		}

		err := i(context.Background(), "/test/Method", nil, nil, nil, invoker)
		require.NoError(t, err)
		assert.Equal(t, "Bearer oauth-test-token", captured)
	})

	t.Run("nil_source", func(t *testing.T) {
		src := interceptor.OAuth2TokenSource(nil)
		_, err := src(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token source is nil")
	})

	t.Run("token_error", func(t *testing.T) {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ""})
		src := interceptor.OAuth2TokenSource(ts)
		_, err := src(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty access token")
	})

	t.Run("token_fetch_error", func(t *testing.T) {
		src := interceptor.OAuth2TokenSource(&failingTokenSource{})
		_, err := src(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "oauth2: failed to get token")
	})
}
