// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor_test

import (
	"context"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/server/interceptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestConfigure_WithLogger(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	l := &stubLogger{}

	// Configure the interceptor package with a logger.
	interceptor.Configure(interceptor.WithLogger(l))

	// Verify by creating an interceptor — it should not panic.
	i := interceptor.Logging()
	assert.NotNil(t, i)
}

func TestConfigure_DefaultLogger(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	// Interceptors should still work, falling back to logging.Default().
	i := interceptor.Recovery()
	assert.NotNil(t, i)
}

func TestConfigure_CustomLoggerUsedByInterceptors(t *testing.T) {
	interceptor.ResetConfig()
	t.Cleanup(interceptor.ResetConfig)

	l := &stubLogger{}
	interceptor.Configure(interceptor.WithLogger(l))

	// Exercise the Logging interceptor which uses the configured logger.
	i := interceptor.Logging()

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.v1.Svc/Method"}

	resp, err := i(context.Background(), "req", info, handler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.True(t, l.called, "custom logger should have been invoked")
}

// stubLogger is a custom logging.Handler that records whether it was called.
type stubLogger struct {
	called bool
}

func (l *stubLogger) Debug(msg string, args ...any) { l.called = true }
func (l *stubLogger) Info(msg string, args ...any)  { l.called = true }
func (l *stubLogger) Warn(msg string, args ...any)  { l.called = true }
func (l *stubLogger) Error(msg string, args ...any) { l.called = true }
