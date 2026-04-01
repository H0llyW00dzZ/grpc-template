// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Tests in this file mutate the global slog default and the package-level
// logging.Default handler. They must NOT use t.Parallel() to avoid
// cross-test interference.
package logging_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault_ReturnsSlogHandler(t *testing.T) {
	h := logging.Default()
	require.NotNil(t, h)
}

func TestSetDefault_OverridesDefaultHandler(t *testing.T) {
	original := logging.Default()
	t.Cleanup(func() { logging.SetDefault(original) })

	stub := &stubHandler{}
	logging.SetDefault(stub)

	assert.Equal(t, stub, logging.Default())
}

func TestSlogHandler_Debug(t *testing.T) {
	out := captureSlog(t, slog.LevelDebug, func(h logging.Handler) {
		h.Debug("debug message", "key", "value")
	})
	assert.Contains(t, out, "debug message")
	assert.Contains(t, out, "key=value")
}

func TestSlogHandler_Info(t *testing.T) {
	out := captureSlog(t, slog.LevelInfo, func(h logging.Handler) {
		h.Info("info message", "key", "value")
	})
	assert.Contains(t, out, "info message")
	assert.Contains(t, out, "key=value")
}

func TestSlogHandler_Warn(t *testing.T) {
	out := captureSlog(t, slog.LevelWarn, func(h logging.Handler) {
		h.Warn("warn message", "key", "value")
	})
	assert.Contains(t, out, "warn message")
	assert.Contains(t, out, "key=value")
}

func TestSlogHandler_Error(t *testing.T) {
	out := captureSlog(t, slog.LevelError, func(h logging.Handler) {
		h.Error("error message", "key", "value")
	})
	assert.Contains(t, out, "error message")
	assert.Contains(t, out, "key=value")
}

func TestSetDefault_NilPanics(t *testing.T) {
	require.Panics(t, func() {
		logging.SetDefault(nil)
	})
}

func TestCustomHandler_ReceivesCalls(t *testing.T) {
	stub := &stubHandler{}
	stub.Debug("d", "k", "v")
	stub.Info("i", "k", "v")
	stub.Warn("w", "k", "v")
	stub.Error("e", "k", "v")

	want := []string{"d", "i", "w", "e"}
	require.Len(t, stub.messages, len(want))
	assert.Equal(t, want, stub.messages)
}

type stubHandler struct {
	messages []string
}

func (s *stubHandler) Debug(msg string, _ ...any) { s.messages = append(s.messages, msg) }
func (s *stubHandler) Info(msg string, _ ...any)  { s.messages = append(s.messages, msg) }
func (s *stubHandler) Warn(msg string, _ ...any)  { s.messages = append(s.messages, msg) }
func (s *stubHandler) Error(msg string, _ ...any) { s.messages = append(s.messages, msg) }

func captureSlog(t *testing.T, level slog.Level, fn func(logging.Handler)) string {
	t.Helper()

	var buf bytes.Buffer
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level})))

	fn(logging.Default())

	return buf.String()
}
