// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package logging_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
)

func TestDefault_ReturnsSlogHandler(t *testing.T) {
	h := logging.Default()
	if h == nil {
		t.Fatal("Default() returned nil")
	}
}

func TestSetDefault_OverridesDefaultHandler(t *testing.T) {
	original := logging.Default()
	t.Cleanup(func() { logging.SetDefault(original) })

	stub := &stubHandler{}
	logging.SetDefault(stub)

	if logging.Default() != stub {
		t.Fatal("SetDefault did not replace the default handler")
	}
}

func TestSlogHandler_Debug(t *testing.T) {
	out := captureSlog(t, slog.LevelDebug, func(h logging.Handler) {
		h.Debug("debug message", "key", "value")
	})
	assertContains(t, out, "debug message")
	assertContains(t, out, "key=value")
}

func TestSlogHandler_Info(t *testing.T) {
	out := captureSlog(t, slog.LevelInfo, func(h logging.Handler) {
		h.Info("info message", "key", "value")
	})
	assertContains(t, out, "info message")
	assertContains(t, out, "key=value")
}

func TestSlogHandler_Warn(t *testing.T) {
	out := captureSlog(t, slog.LevelWarn, func(h logging.Handler) {
		h.Warn("warn message", "key", "value")
	})
	assertContains(t, out, "warn message")
	assertContains(t, out, "key=value")
}

func TestSlogHandler_Error(t *testing.T) {
	out := captureSlog(t, slog.LevelError, func(h logging.Handler) {
		h.Error("error message", "key", "value")
	})
	assertContains(t, out, "error message")
	assertContains(t, out, "key=value")
}

func TestCustomHandler_ReceivesCalls(t *testing.T) {
	stub := &stubHandler{}
	stub.Debug("d", "k", "v")
	stub.Info("i", "k", "v")
	stub.Warn("w", "k", "v")
	stub.Error("e", "k", "v")

	want := []string{"d", "i", "w", "e"}
	if len(stub.messages) != len(want) {
		t.Fatalf("got %d messages, want %d", len(stub.messages), len(want))
	}
	for i, msg := range want {
		if stub.messages[i] != msg {
			t.Errorf("message[%d] = %q, want %q", i, stub.messages[i], msg)
		}
	}
}

// ---------- helpers ----------

// stubHandler records messages for assertion.
type stubHandler struct {
	messages []string
}

func (s *stubHandler) Debug(msg string, _ ...any) { s.messages = append(s.messages, msg) }
func (s *stubHandler) Info(msg string, _ ...any)  { s.messages = append(s.messages, msg) }
func (s *stubHandler) Warn(msg string, _ ...any)  { s.messages = append(s.messages, msg) }
func (s *stubHandler) Error(msg string, _ ...any) { s.messages = append(s.messages, msg) }

// captureSlog redirects slog output to a buffer for the duration of fn,
// then returns the captured text.
func captureSlog(t *testing.T, level slog.Level, fn func(logging.Handler)) string {
	t.Helper()

	var buf bytes.Buffer
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level})))

	fn(logging.Default())

	return buf.String()
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("output %q does not contain %q", s, substr)
	}
}
