// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

// healthWatchFunc creates a health Watch stream for the given connection.
// The default implementation uses the standard gRPC health client.
type healthWatchFunc func(ctx context.Context, conn *grpc.ClientConn) (healthgrpc.Health_WatchClient, error)

// Client wraps a gRPC client connection with lifecycle management
// and functional options configuration. It is the client-side
// counterpart of [github.com/H0llyW00dzZ/grpc-template/internal/server.Server].
type Client struct {
	target             string
	insecureCreds      bool
	tlsConfig          *tls.Config
	logger             logging.Handler
	unaryInterceptors  []grpc.UnaryClientInterceptor
	streamInterceptors []grpc.StreamClientInterceptor
	dialOpts           []grpc.DialOption
	conn               *grpc.ClientConn
	healthWatch        bool
	healthCancel       context.CancelFunc
	mu                 sync.RWMutex

	// configErr captures errors from functional options (e.g., TLS
	// certificate loading) so they can be returned from [Client.Connect]
	// instead of panicking during construction.
	configErr error

	// dialFunc overrides grpc.NewClient for testing. If nil, the real
	// grpc.NewClient is used.
	dialFunc func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)

	// watchFunc overrides the health Watch creation for testing.
	// If nil, the standard gRPC health client is used.
	watchFunc healthWatchFunc
}

// New creates a new Client targeting the given address.
// The connection is not established until [Client.Connect] is called.
//
//	c := client.New("localhost:50051",
//	    client.WithInsecure(),
//	    client.WithLogger(myLogger),
//	)
func New(target string, opts ...Option) *Client {
	c := &Client{
		target: target,
		logger: logging.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Conn returns the underlying [grpc.ClientConn] for creating service stubs.
//
//	greeter := pb.NewGreeterServiceClient(c.Conn())
//
// Conn panics if called before a successful [Client.Connect].
func (c *Client) Conn() *grpc.ClientConn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conn == nil {
		panic("client: Conn called before Connect")
	}
	return c.conn
}

// Logger returns the configured logger.
func (c *Client) Logger() logging.Handler {
	return c.logger
}

// Connect establishes the gRPC connection using the configured options.
// The connection is created lazily via [grpc.NewClient]; the actual TCP
// handshake occurs when the first RPC is made or [Client.WaitReady] is
// called.
//
// If [WithHealthWatch] was configured, a background goroutine is started
// to monitor the server's health status.
func (c *Client) Connect(ctx context.Context) error {
	if c.configErr != nil {
		return fmt.Errorf("client: configuration error: %w", c.configErr)
	}

	opts := c.buildDialOpts()

	dial := grpc.NewClient
	if c.dialFunc != nil {
		dial = c.dialFunc
	}

	conn, err := dial(c.target, opts...)
	if err != nil {
		return fmt.Errorf("client: failed to create connection to %s: %w", c.target, err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	c.logger.Info("gRPC client connected", "target", c.target)

	if c.healthWatch {
		c.startHealthWatch()
	}

	return nil
}

// WaitReady blocks until the client connection reaches the Ready state
// or the context expires. Use this after [Client.Connect] when you need
// to ensure the server is reachable before sending RPCs.
//
//	if err := c.Connect(ctx); err != nil { ... }
//	if err := c.WaitReady(ctx); err != nil { ... }
func (c *Client) WaitReady(ctx context.Context) error {
	conn := c.Conn()
	conn.Connect()

	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if !conn.WaitForStateChange(ctx, state) {
			return fmt.Errorf("client: context expired waiting for ready state: %w", ctx.Err())
		}
	}
}

// State returns the current connectivity state of the underlying connection.
// Returns [connectivity.Shutdown] if the client has not been connected.
func (c *Client) State() connectivity.State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conn == nil {
		return connectivity.Shutdown
	}
	return c.conn.GetState()
}

// Close gracefully shuts down the gRPC client connection and cancels
// any background goroutines (e.g., health watching).
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.healthCancel != nil {
		c.healthCancel()
		c.healthCancel = nil
	}

	if c.conn != nil {
		c.logger.Info("gRPC client closing", "target", c.target)
		err := c.conn.Close()
		c.conn = nil
		return err
	}

	return nil
}

func (c *Client) buildDialOpts() []grpc.DialOption {
	var opts []grpc.DialOption

	switch {
	case c.tlsConfig != nil:
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(c.tlsConfig)))
	case c.insecureCreds:
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	default:
		// Default to insecure for template convenience; warn so
		// production deployments don't silently skip TLS.
		c.logger.Warn("no TLS configured, using insecure credentials — do not use in production",
			"target", c.target)
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if len(c.unaryInterceptors) > 0 {
		opts = append(opts, grpc.WithChainUnaryInterceptor(c.unaryInterceptors...))
	}

	if len(c.streamInterceptors) > 0 {
		opts = append(opts, grpc.WithChainStreamInterceptor(c.streamInterceptors...))
	}

	opts = append(opts, c.dialOpts...)

	return opts
}

func (c *Client) startHealthWatch() {
	ctx, cancel := context.WithCancel(context.Background())

	c.mu.Lock()
	c.healthCancel = cancel
	conn := c.conn
	c.mu.Unlock()

	go func() {
		watchFn := c.watchFunc
		if watchFn == nil {
			watchFn = defaultHealthWatch
		}

		const (
			initialBackoff = 500 * time.Millisecond
			maxBackoff     = 30 * time.Second
		)
		backoff := initialBackoff

		for {
			stream, err := watchFn(ctx, conn)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Warn("health watch failed to start, retrying",
					"target", c.target, "backoff", backoff, "error", err)
				if !c.sleepOrDone(ctx, backoff) {
					return
				}
				backoff = min(backoff*2, maxBackoff)
				continue
			}

			// Reset backoff on successful connection.
			backoff = initialBackoff

			for {
				resp, err := stream.Recv()
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					c.logger.Warn("health watch stream ended, reconnecting",
						"target", c.target, "backoff", backoff, "error", err)
					break
				}
				c.logger.Info("health status changed",
					"target", c.target,
					"status", resp.GetStatus().String(),
				)
			}

			// Wait before reconnecting after a stream error.
			if !c.sleepOrDone(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
		}
	}()
}

// sleepOrDone blocks for the given duration or until the context is cancelled.
// Returns true if the sleep completed, false if the context was cancelled.
func (c *Client) sleepOrDone(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// defaultHealthWatch creates a health Watch stream using the standard
// gRPC Health Checking Protocol client.
func defaultHealthWatch(ctx context.Context, conn *grpc.ClientConn) (healthgrpc.Health_WatchClient, error) {
	return healthgrpc.NewHealthClient(conn).Watch(ctx, &healthgrpc.HealthCheckRequest{})
}
