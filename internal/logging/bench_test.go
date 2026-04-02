// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package logging_test

import (
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
)

// BenchmarkDefault measures the cost of loading the default logger
// via atomic.Value, which is the fallback path for every interceptor.
func BenchmarkDefault(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = logging.Default()
	}
}

// BenchmarkSetDefault measures the cost of replacing the global logger.
// This is only called during startup and is not on the hot path.
func BenchmarkSetDefault(b *testing.B) {
	l := logging.Default()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		logging.SetDefault(l)
	}
}
