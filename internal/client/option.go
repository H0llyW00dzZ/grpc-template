// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/client/interceptor"
	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
)

// Option is a functional option for configuring the [Client].
type Option func(*Client)

// WithInsecure disables transport security.
// Use only for development and testing environments.
func WithInsecure() Option {
	return func(c *Client) {
		c.insecureCreds = true
	}
}

// WithTLS enables TLS on the client using the given CA certificate file
// to verify the server's identity. If the certificate cannot be read or
// parsed, the error is deferred and returned from [Client.Connect].
//
//	c := client.New("example.com:443",
//	    client.WithTLS("/path/to/ca.pem"),
//	)
func WithTLS(caCertFile string) Option {
	return func(c *Client) {
		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			c.configErr = fmt.Errorf("failed to read CA certificate: %w", err)
			return
		}

		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			c.configErr = fmt.Errorf("failed to parse CA certificate from %s", caCertFile)
			return
		}

		c.configErr = nil
		c.tlsConfig = &tls.Config{
			RootCAs:    caPool,
			MinVersion: tls.VersionTLS13,
		}
	}
}

// WithMutualTLS enables mutual TLS (mTLS) on the client.
// The client presents its own certificate and verifies the server's
// certificate against the given CA. Use this for zero-trust environments
// and service-to-service communication. If any certificate cannot be
// loaded or parsed, the error is deferred and returned from [Client.Connect].
func WithMutualTLS(certFile, keyFile, caCertFile string) Option {
	return func(c *Client) {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			c.configErr = fmt.Errorf("failed to load client TLS certificate: %w", err)
			return
		}

		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			c.configErr = fmt.Errorf("failed to read CA certificate: %w", err)
			return
		}

		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			c.configErr = fmt.Errorf("failed to parse CA certificate from %s", caCertFile)
			return
		}

		c.configErr = nil
		c.tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caPool,
			MinVersion:   tls.VersionTLS13,
		}
	}
}

// WithLogger sets the logger used by the client and its interceptors.
// It also calls [interceptor.Configure] with [interceptor.WithLogger]
// so that all client interceptors automatically use the same logger.
func WithLogger(l logging.Handler) Option {
	return func(c *Client) {
		if l != nil {
			c.logger = l
			interceptor.Configure(interceptor.WithLogger(l))
		}
	}
}

// WithUnaryInterceptors appends unary client interceptors to the chain.
func WithUnaryInterceptors(interceptors ...grpc.UnaryClientInterceptor) Option {
	return func(c *Client) {
		c.unaryInterceptors = append(c.unaryInterceptors, interceptors...)
	}
}

// WithStreamInterceptors appends stream client interceptors to the chain.
func WithStreamInterceptors(interceptors ...grpc.StreamClientInterceptor) Option {
	return func(c *Client) {
		c.streamInterceptors = append(c.streamInterceptors, interceptors...)
	}
}

// WithKeepalive sets the keepalive parameters for the client connection.
// This is important for long-lived connections that may traverse load
// balancers or firewalls that drop idle connections.
func WithKeepalive(params keepalive.ClientParameters) Option {
	return func(c *Client) {
		c.dialOpts = append(c.dialOpts, grpc.WithKeepaliveParams(params))
	}
}

// WithMaxMsgSize overrides the default 4MB message size limit for both
// receiving and sending messages.
func WithMaxMsgSize(maxBytes int) Option {
	return func(c *Client) {
		c.dialOpts = append(c.dialOpts,
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(maxBytes),
				grpc.MaxCallSendMsgSize(maxBytes),
			),
		)
	}
}

// WithDialOptions allows safely passing any other raw [grpc.DialOption]
// to the underlying client connection.
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(c *Client) {
		c.dialOpts = append(c.dialOpts, opts...)
	}
}

// WithHealthWatch enables background health status monitoring.
// After [Client.Connect], a goroutine watches the server's health
// status via the standard gRPC Health Checking Protocol and logs changes.
// The watch is automatically cancelled by [Client.Close].
func WithHealthWatch() Option {
	return func(c *Client) {
		c.healthWatch = true
	}
}

// WithDefaultTimeout configures the default RPC deadline applied by
// the [interceptor.Timeout] interceptor.
// It delegates to [interceptor.Configure] with [interceptor.WithDefaultTimeout].
func WithDefaultTimeout(d time.Duration) Option {
	return func(c *Client) {
		interceptor.Configure(interceptor.WithDefaultTimeout(d))
	}
}

// WithRetry configures the retry interceptor's maximum attempts and
// base backoff duration.
// It delegates to [interceptor.Configure] with [interceptor.WithRetry].
//
//	c := client.New("localhost:50051",
//	    client.WithRetry(3, time.Second), // up to 3 retries, 1s base backoff
//	)
func WithRetry(maxRetries int, backoff time.Duration) Option {
	return func(c *Client) {
		interceptor.Configure(interceptor.WithRetry(maxRetries, backoff))
	}
}

// WithRetryCodes overrides the default set of retryable gRPC status codes.
// It delegates to [interceptor.Configure] with [interceptor.WithRetryCodes].
func WithRetryCodes(retryCodes ...codes.Code) Option {
	return func(c *Client) {
		interceptor.Configure(interceptor.WithRetryCodes(retryCodes...))
	}
}

// WithTokenSource configures bearer token injection for the
// [interceptor.Auth] and [interceptor.StreamAuth] interceptors.
// It delegates to [interceptor.Configure] with [interceptor.WithTokenSource].
//
// Use [interceptor.StaticToken] for static tokens or [interceptor.OAuth2TokenSource]
// for dynamic OAuth2 (with automatic refresh).
//
//	c := client.New("localhost:50051",
//	    client.WithTokenSource(interceptor.StaticToken("my-api-key")),
//	    // client.WithTokenSource(interceptor.OAuth2TokenSource(oauth2Src)),
//	)
func WithTokenSource(fn interceptor.TokenSource) Option {
	return func(c *Client) {
		interceptor.Configure(interceptor.WithTokenSource(fn))
	}
}
