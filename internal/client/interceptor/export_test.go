// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package interceptor

import (
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
)

// ResetConfig resets the package-level configuration to defaults.
// This is only available in tests.
func ResetConfig() {
	configMu.Lock()
	defer configMu.Unlock()
	defaultConfig = &config{
		retryCodes: defaultRetryCodes,
	}
}

// TestableLogger returns the configured logger for testing,
// falling back to [logging.Default] if none has been set.
func TestableLogger() logging.Handler {
	return logging.Resolve(getConfig().logger)
}

// BackoffDuration exports backoffDuration for testing.
func BackoffDuration(attempt int, base time.Duration) time.Duration {
	return backoffDuration(attempt, base)
}
