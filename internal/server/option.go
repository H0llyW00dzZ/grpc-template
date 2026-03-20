// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

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
func WithTLS(certFile, keyFile string) Option {
	return func(s *Server) {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			panic(fmt.Sprintf("failed to load TLS certificate: %v", err))
		}
		s.tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS13,
		}
	}
}

// WithMutualTLS enables mutual TLS (mTLS) on the server.
// Both the server and client verify each other's certificates.
// Use this for zero-trust environments and service-to-service communication.
func WithMutualTLS(certFile, keyFile, caCertFile string) Option {
	return func(s *Server) {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			panic(fmt.Sprintf("failed to load TLS certificate: %v", err))
		}

		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			panic(fmt.Sprintf("failed to read CA certificate: %v", err))
		}

		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			panic("failed to parse CA certificate")
		}

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
