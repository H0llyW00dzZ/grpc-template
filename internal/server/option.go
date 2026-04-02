// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/H0llyW00dzZ/grpc-template/internal/server/interceptor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// Option is a functional option for configuring the Server.
type Option func(*Server)

// WithPort sets the TCP port the server listens on.
// Default is "50051".
func WithPort(port string) Option {
	return func(s *Server) {
		s.port = port
	}
}

// WithReflection enables gRPC server reflection.
// This is useful for debugging with tools like grpcurl.
func WithReflection() Option {
	return func(s *Server) {
		s.reflection = true
	}
}

// WithTLS enables TLS on the server using the given certificate and key files.
// This encrypts all connections between clients and the server.
// If the certificate cannot be loaded, the error is deferred and
// returned from [Server.Run].
func WithTLS(certFile, keyFile string) Option {
	return func(s *Server) {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			s.configErr = fmt.Errorf("failed to load TLS certificate: %w", err)
			s.tlsConfig = nil
			return
		}
		s.configErr = nil
		s.tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS13,
		}
	}
}

// WithMutualTLS enables mutual TLS (mTLS) on the server.
// Both the server and client verify each other's certificates.
// Use this for zero-trust environments and service-to-service communication.
// If any certificate cannot be loaded or parsed, the error is deferred
// and returned from [Server.Run].
func WithMutualTLS(certFile, keyFile, caCertFile string) Option {
	return func(s *Server) {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			s.configErr = fmt.Errorf("failed to load TLS certificate: %w", err)
			s.tlsConfig = nil
			return
		}

		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			s.configErr = fmt.Errorf("failed to read CA certificate: %w", err)
			s.tlsConfig = nil
			return
		}

		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			s.configErr = fmt.Errorf("failed to parse CA certificate from %s", caCertFile)
			s.tlsConfig = nil
			return
		}

		s.configErr = nil
		s.tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    caPool,
			MinVersion:   tls.VersionTLS13,
		}
	}
}

// WithUnaryInterceptors appends unary server interceptors.
func WithUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) Option {
	return func(s *Server) {
		s.unaryInterceptors = append(s.unaryInterceptors, interceptors...)
	}
}

// WithStreamInterceptors appends stream server interceptors.
func WithStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) Option {
	return func(s *Server) {
		s.streamInterceptors = append(s.streamInterceptors, interceptors...)
	}
}

// WithKeepalive sets the keepalive parameters and enforcement policy.
// This is crucial for long-lived streams to prevent load balancers or firewalls from dropping idle connections.
func WithKeepalive(params keepalive.ServerParameters, policy keepalive.EnforcementPolicy) Option {
	return func(s *Server) {
		s.grpcOpts = append(s.grpcOpts, grpc.KeepaliveParams(params), grpc.KeepaliveEnforcementPolicy(policy))
	}
}

// WithMaxMsgSize overrides the default 4MB message size limit for both receiving and sending messages.
// Useful for services handling large payloads like Key-Value pairs or media metadata.
func WithMaxMsgSize(maxBytes int) Option {
	return func(s *Server) {
		s.grpcOpts = append(s.grpcOpts, grpc.MaxRecvMsgSize(maxBytes), grpc.MaxSendMsgSize(maxBytes))
	}
}

// WithMaxConcurrentStreams limits the number of concurrent streams to each ServerTransport.
// Useful to protect the server from being overwhelmed by bursty traffic.
func WithMaxConcurrentStreams(n uint32) Option {
	return func(s *Server) {
		s.grpcOpts = append(s.grpcOpts, grpc.MaxConcurrentStreams(n))
	}
}

// WithGrpcOptions allows safely passing any other raw grpc.ServerOption to the underlying server.
func WithGrpcOptions(opts ...grpc.ServerOption) Option {
	return func(s *Server) {
		s.grpcOpts = append(s.grpcOpts, opts...)
	}
}

// WithListener sets a pre-created net.Listener for the server to use
// instead of opening a new TCP listener on the configured port.
// This is useful for testing (e.g., bufconn) and custom deployment
// scenarios such as Unix domain sockets or systemd socket activation.
//
// The listener is consumed by the first [Server.Run] call; subsequent
// Run calls on the same Server fall back to the configured port.
func WithListener(lis net.Listener) Option {
	return func(s *Server) {
		s.listener = lis
	}
}

// WithLogger sets the logger used by the gRPC server and its interceptors.
// It also calls [interceptor.Configure] with [interceptor.WithLogger] so that
// all interceptors in the package automatically use the same logger.
func WithLogger(l logging.Handler) Option {
	return func(s *Server) {
		if l != nil {
			s.logger = l
			logging.SetDefault(l)
			interceptor.Configure(interceptor.WithLogger(l))
		}
	}
}

// WithAuthFunc sets the authentication function used by interceptors.
// It delegates to [interceptor.Configure] with [interceptor.WithAuthFunc].
func WithAuthFunc(fn interceptor.AuthFunc) Option {
	return func(s *Server) {
		interceptor.Configure(interceptor.WithAuthFunc(fn))
	}
}

// WithExcludedMethods configures the auth interceptor to skip authentication
// for the given fully-qualified gRPC method names.
// It delegates to [interceptor.Configure] with [interceptor.WithExcludedMethods].
func WithExcludedMethods(methods ...string) Option {
	return func(s *Server) {
		interceptor.Configure(interceptor.WithExcludedMethods(methods...))
	}
}

// WithRateLimit sets the per-peer rate limit in requests per second and
// the burst size (maximum number of requests allowed at once).
// It uses the default in-memory token-bucket limiter.
//
// If [WithRateLimiter] is also used, whichever is applied last takes effect
// because both options write to the same underlying rate limiter field.
//
// It delegates to [interceptor.Configure] with [interceptor.WithRateLimit].
//
//	srv := server.New(
//	    server.WithRateLimit(100, 200), // 100 req/s, burst up to 200
//	)
func WithRateLimit(rps float64, burst int) Option {
	return func(s *Server) {
		interceptor.Configure(interceptor.WithRateLimit(rps, burst))
	}
}

// WithRateLimiter sets a custom [interceptor.RateLimiter] implementation
// for per-peer rate limiting. Use this to plug in a distributed backend
// such as Redis instead of the default in-memory limiter.
//
// When set, this overrides any limiter configured via [WithRateLimit],
// since both options write to the same underlying rate limiter field.
// The rate and burst parameters are owned by the custom implementation.
//
//	srv := server.New(
//	    server.WithRateLimiter(myRedisLimiter), // replaces the default in-memory limiter
//	)
//
// It delegates to [interceptor.Configure] with [interceptor.WithRateLimiter].
func WithRateLimiter(l interceptor.RateLimiter) Option {
	return func(s *Server) {
		interceptor.Configure(interceptor.WithRateLimiter(l))
	}
}

// WithTrustProxy enables trust for proxy headers (X-Forwarded-For, X-Real-IP)
// when extracting client IPs for per-peer rate limiting.
//
// WARNING: Only enable this when your server is behind a trusted reverse proxy
// or load balancer that sanitizes these headers. Without a trusted proxy,
// clients can spoof their IP address to bypass rate limits.
//
// It delegates to [interceptor.Configure] with [interceptor.WithTrustProxy].
func WithTrustProxy(trust bool) Option {
	return func(s *Server) {
		interceptor.Configure(interceptor.WithTrustProxy(trust))
	}
}
