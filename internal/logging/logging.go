// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package logging defines a minimal Handler interface for structured logging.
//
// The interface covers the four standard severity levels (Debug, Info, Warn,
// Error) and uses the same variadic key-value signature as [log/slog], making
// the default slog-backed implementation a zero-cost wrapper.
//
// To swap in a different backend (zap, zerolog, logrus, …), implement the
// [Handler] interface and pass it to interceptors and services.
//
// # Default Handler
//
// [Default] returns a slog-backed handler.  Override the global default with
// [SetDefault]. Both functions are safe for concurrent use:
//
//	logging.SetDefault(myZapAdapter)
package logging

import (
	"log/slog"
	"sync/atomic"
)

// Handler is the interface for structured logging at four severity levels.
// Each method accepts a message and alternating key-value pairs, following
// the same convention as [log/slog].
type Handler interface {
	// Debug logs at DEBUG level.
	Debug(msg string, args ...any)
	// Info logs at INFO level.
	Info(msg string, args ...any)
	// Warn logs at WARN level.
	Warn(msg string, args ...any)
	// Error logs at ERROR level.
	Error(msg string, args ...any)
}

// handlerBox wraps a [Handler] so that [atomic.Value] always stores the
// same concrete type, regardless of which Handler implementation is inside.
type handlerBox struct{ h Handler }

// defaultHandler is the package-level default handler, stored atomically
// so that concurrent calls to [Default] and [SetDefault] are safe.
var defaultHandler atomic.Value

func init() { defaultHandler.Store(handlerBox{h: &slogHandler{}}) }

// Default returns the current default Handler (slog-backed unless overridden).
// It is safe for concurrent use.
func Default() Handler { return defaultHandler.Load().(handlerBox).h }

// SetDefault replaces the package-level default Handler.
// It panics if h is nil to fail fast rather than causing a nil-pointer
// dereference on the next call to [Default].
// It is safe for concurrent use.
func SetDefault(h Handler) {
	if h == nil {
		panic("logging: SetDefault called with nil Handler")
	}
	defaultHandler.Store(handlerBox{h: h})
}

// slogHandler adapts [log/slog] to the [Handler] interface.
type slogHandler struct{}

func (*slogHandler) Debug(msg string, args ...any) { slog.Debug(msg, args...) }
func (*slogHandler) Info(msg string, args ...any)  { slog.Info(msg, args...) }
func (*slogHandler) Warn(msg string, args ...any)  { slog.Warn(msg, args...) }
func (*slogHandler) Error(msg string, args ...any) { slog.Error(msg, args...) }
