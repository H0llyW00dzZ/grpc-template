// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package client

import (
	"context"

	"google.golang.org/grpc"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

// SetDialFunc overrides the gRPC dial function for testing.
// Pass nil to restore the default [grpc.NewClient].
//
// Must be called before [Client.Connect]; these fields are treated
// as immutable once the connection lifecycle begins.
func (c *Client) SetDialFunc(fn func(string, ...grpc.DialOption) (*grpc.ClientConn, error)) {
	c.dialFunc = fn
}

// SetWatchFunc overrides the health Watch creation for testing.
// Pass nil to restore the default health client.
//
// Must be called before [Client.Connect]; these fields are treated
// as immutable once the connection lifecycle begins.
func (c *Client) SetWatchFunc(fn func(context.Context, *grpc.ClientConn) (healthgrpc.Health_WatchClient, error)) {
	c.watchFunc = fn
}

// SetListFunc overrides the list services function for testing.
// Pass nil to restore the default reflection-based implementation.
func (c *Client) SetListFunc(fn func(context.Context, *grpc.ClientConn) ([]string, error)) {
	c.listFunc = fn
}

// DefaultListServices exposes the default reflection-based implementation
// for testing its error branches.
var DefaultListServices = defaultListServices
