// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os/signal"
	"syscall"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// ServiceRegistrar is a function that registers gRPC services on a server.
// Use this to decouple service implementations from the server package.
type ServiceRegistrar func(*grpc.Server)

// Server wraps a gRPC server with lifecycle management.
type Server struct {
	port               string
	reflection         bool
	tlsConfig          *tls.Config
	logger             logging.Handler
	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor
	registrars         []ServiceRegistrar
	grpcOpts           []grpc.ServerOption
	listener           net.Listener
	healthSrv          *health.Server

	// configErr captures errors from functional options (e.g., TLS
	// certificate loading) so they can be returned from [Server.Run]
	// instead of panicking during construction.
	configErr error
}

// New creates a new Server with the given functional options.
func New(opts ...Option) *Server {
	s := &Server{
		port:   "50051",
		logger: logging.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Logger returns the configured logger for the server.
func (s *Server) Logger() logging.Handler {
	return s.logger
}

// Health returns the gRPC health server, allowing callers to toggle
// per-service serving status at runtime (e.g., for maintenance).
//
//	// Mark a service as temporarily unavailable:
//	srv.Health().SetServingStatus(
//	    "helloworld.v1.GreeterService",
//	    healthgrpc.HealthCheckResponse_NOT_SERVING,
//	)
//
//	// Restore it:
//	srv.Health().SetServingStatus(
//	    "helloworld.v1.GreeterService",
//	    healthgrpc.HealthCheckResponse_SERVING,
//	)
func (s *Server) Health() *health.Server {
	return s.healthSrv
}

// RegisterService adds one or more service registrars that will be called
// when the server starts. This is the primary way to add your
// gRPC service implementations to the server.
//
//	srv.RegisterService(greeterSvc.Register, authSvc.Register, kvSvc.Register)
func (s *Server) RegisterService(registrars ...ServiceRegistrar) {
	s.registrars = append(s.registrars, registrars...)
}

func (s *Server) buildOptions() []grpc.ServerOption {
	var opts []grpc.ServerOption

	if s.tlsConfig != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(s.tlsConfig)))
		s.logger.Info("gRPC TLS enabled")
	}

	if len(s.unaryInterceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(s.unaryInterceptors...))
	}

	if len(s.streamInterceptors) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(s.streamInterceptors...))
	}

	if len(s.grpcOpts) > 0 {
		opts = append(opts, s.grpcOpts...)
	}

	return opts
}

func (s *Server) setupServer() *grpc.Server {
	opts := s.buildOptions()
	grpcServer := grpc.NewServer(opts...)

	for _, registrar := range s.registrars {
		registrar(grpcServer)
	}

	s.healthSrv = health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, s.healthSrv)
	s.healthSrv.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)

	if s.reflection {
		reflection.Register(grpcServer)
		s.logger.Info("gRPC server reflection enabled")
	}

	return grpcServer
}

// Run starts the gRPC server and blocks until the context is cancelled
// or an OS interrupt/termination signal is received.
//
// It returns an error if any functional option (e.g., [WithTLS],
// [WithMutualTLS]) recorded a configuration error during construction.
//
// It performs a graceful shutdown, allowing in-flight RPCs to complete.
func (s *Server) Run(ctx context.Context) error {
	if s.configErr != nil {
		return fmt.Errorf("server: configuration error: %w", s.configErr)
	}

	// Listen for OS signals for graceful shutdown.
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	grpcServer := s.setupServer()

	// Use injected listener or start a new TCP listener.
	lis := s.listener
	if lis == nil {
		var err error
		lis, err = net.Listen("tcp", ":"+s.port)
		if err != nil {
			return fmt.Errorf("failed to listen on port %s: %w", s.port, err)
		}
	}

	// Start serving in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("gRPC server listening", "port", s.port)
		if err := grpcServer.Serve(lis); err != nil {
			errCh <- fmt.Errorf("failed to serve: %w", err)
		}
		close(errCh)
	}()

	// Wait for shutdown signal or serve error.
	select {
	case <-ctx.Done():
		s.logger.Info("shutdown signal received, draining connections...")
		s.healthSrv.SetServingStatus("", healthgrpc.HealthCheckResponse_NOT_SERVING)
		grpcServer.GracefulStop()
		s.logger.Info("gRPC server stopped gracefully")
		return nil
	case err := <-errCh:
		return err
	}
}
