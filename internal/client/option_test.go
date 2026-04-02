// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package client_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/H0llyW00dzZ/grpc-template/internal/client"
	_ "github.com/H0llyW00dzZ/grpc-template/internal/client/balancer"
	clientinterceptor "github.com/H0llyW00dzZ/grpc-template/internal/client/interceptor"
	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
)

func TestWithInsecure(t *testing.T) {
	c := client.New("localhost:50051", client.WithInsecure())
	require.NotNil(t, c)
}

func TestWithInsecure_ClearsTLSError(t *testing.T) {
	// A preceding TLS error should be cleared by WithInsecure.
	c := client.New("localhost:50051",
		client.WithMutualTLS("/bad/cert.pem", "/bad/key.pem", "/bad/ca.pem"),
		client.WithInsecure(),
	)
	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
}

func TestWithTLS(t *testing.T) {
	dir := t.TempDir()
	_, _, caCertFile := generateTestCert(t, dir)

	c := client.New("localhost:50051", client.WithTLS(caCertFile))
	require.NotNil(t, c)
}

func TestWithTLS_InvalidFile(t *testing.T) {
	c := client.New("localhost:50051", client.WithTLS("/no/such/ca.pem"))
	err := c.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration error")
	assert.Contains(t, err.Error(), "failed to read CA certificate")
}

func TestWithTLS_SuccessOverridesPreviousError(t *testing.T) {
	dir := t.TempDir()
	_, _, caCertFile := generateTestCert(t, dir)

	// A failing option followed by a succeeding one should connect
	// without error because the second option clears configErr.
	c := client.New("localhost:50051",
		client.WithTLS("/no/such/ca.pem"),
		client.WithTLS(caCertFile),
	)
	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
}

func TestWithTLS_InvalidPEM(t *testing.T) {
	dir := t.TempDir()
	badCA := filepath.Join(dir, "bad_ca.pem")
	require.NoError(t, os.WriteFile(badCA, []byte("not a PEM"), 0o600))

	c := client.New("localhost:50051", client.WithTLS(badCA))
	err := c.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration error")
	assert.Contains(t, err.Error(), "failed to parse CA certificate")
}

func TestWithMutualTLS(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, caCertFile := generateTestCert(t, dir)

	c := client.New("localhost:50051", client.WithMutualTLS(certFile, keyFile, caCertFile))
	require.NotNil(t, c)
}

func TestWithMutualTLS_InvalidCert(t *testing.T) {
	c := client.New("localhost:50051", client.WithMutualTLS("/bad/cert.pem", "/bad/key.pem", "/bad/ca.pem"))
	err := c.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration error")
	assert.Contains(t, err.Error(), "failed to load client TLS certificate")
}

func TestWithMutualTLS_InvalidCA(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _ := generateTestCert(t, dir)

	c := client.New("localhost:50051", client.WithMutualTLS(certFile, keyFile, "/no/such/ca.pem"))
	err := c.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration error")
	assert.Contains(t, err.Error(), "failed to read CA certificate")
}

func TestWithMutualTLS_InvalidCAPEM(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _ := generateTestCert(t, dir)

	badCA := filepath.Join(dir, "bad_ca.pem")
	require.NoError(t, os.WriteFile(badCA, []byte("not a PEM"), 0o600))

	c := client.New("localhost:50051", client.WithMutualTLS(certFile, keyFile, badCA))
	err := c.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration error")
	assert.Contains(t, err.Error(), "failed to parse CA certificate")
}

func TestWithLogger(t *testing.T) {
	l := logging.Default()
	c := client.New("localhost:50051", client.WithLogger(l))
	require.NotNil(t, c)
	assert.Equal(t, l, c.Logger())
}

func TestWithLogger_Nil(t *testing.T) {
	c := client.New("localhost:50051", client.WithLogger(nil))
	assert.Equal(t, logging.Default(), c.Logger())
}

func TestWithUnaryInterceptors(t *testing.T) {
	noop := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(ctx, method, req, reply, cc, opts...)
	}
	c := client.New("localhost:50051", client.WithUnaryInterceptors(noop))
	require.NotNil(t, c)
}

func TestWithStreamInterceptors(t *testing.T) {
	noop := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return streamer(ctx, desc, cc, method, opts...)
	}
	c := client.New("localhost:50051", client.WithStreamInterceptors(noop))
	require.NotNil(t, c)
}

func TestWithKeepalive(t *testing.T) {
	c := client.New("localhost:50051",
		client.WithKeepalive(keepalive.ClientParameters{
			Time:    30 * time.Second,
			Timeout: 10 * time.Second,
		}),
	)
	require.NotNil(t, c)
}

func TestWithMaxMsgSize(t *testing.T) {
	c := client.New("localhost:50051", client.WithMaxMsgSize(8*1024*1024))
	require.NotNil(t, c)
}

func TestWithDialOptions(t *testing.T) {
	c := client.New("localhost:50051", client.WithDialOptions(grpc.WithUserAgent("test/1.0")))
	require.NotNil(t, c)
}

func TestWithHealthWatch(t *testing.T) {
	c := client.New("localhost:50051", client.WithHealthWatch())
	require.NotNil(t, c)
}

func TestWithDefaultTimeout(t *testing.T) {
	c := client.New("localhost:50051", client.WithDefaultTimeout(5*time.Second))
	require.NotNil(t, c)
}

func TestWithRetry(t *testing.T) {
	c := client.New("localhost:50051", client.WithRetry(3, time.Second))
	require.NotNil(t, c)
}

func TestWithRetryCodes(t *testing.T) {
	c := client.New("localhost:50051", client.WithRetryCodes(codes.Unavailable, codes.Aborted))
	require.NotNil(t, c)
}

func TestWithTokenSource(t *testing.T) {
	c := client.New("localhost:50051",
		client.WithTokenSource(clientinterceptor.StaticToken("my-token")),
	)
	require.NotNil(t, c)
}

func TestWithLoadBalancing(t *testing.T) {
	policies := []string{
		"pick_first",
		"round_robin",
		"weighted_round_robin",
		"least_request_experimental",
		"ring_hash_experimental",
	}
	for _, policy := range policies {
		t.Run(policy, func(t *testing.T) {
			c := client.New("localhost:50051", client.WithLoadBalancing(policy))
			require.NotNil(t, c)
		})
	}
}

func TestWithLoadBalancing_Empty(t *testing.T) {
	c := client.New("localhost:50051", client.WithLoadBalancing(""))
	require.NotNil(t, c)
}

func TestWithLoadBalancing_InvalidPolicy(t *testing.T) {
	c := client.New("localhost:50051",
		client.WithInsecure(),
		client.WithLoadBalancing("banana"),
	)
	err := c.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration error")
	assert.Contains(t, err.Error(), "unknown load balancing policy")
}

func TestWithLoadBalancing_SuccessOverridesPreviousError(t *testing.T) {
	// A failing option followed by a succeeding one should connect
	// without error because the second option clears configErr.
	c := client.New("localhost:50051",
		client.WithInsecure(),
		client.WithLoadBalancing("banana"),
		client.WithLoadBalancing("round_robin"),
	)
	err := c.Connect(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
}
