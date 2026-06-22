package grpcserver

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/zchelalo/neuraclinic-auth/gen/go/auth/v1"
	"github.com/zchelalo/neuraclinic-auth/internal/bootstrap"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
}

func New(cfg bootstrap.Config, logger *zap.Logger, authService authv1.AuthServiceServer) (*Server, error) {
	cert, err := tls.LoadX509KeyPair(cfg.GRPCTLSCertPath, cfg.GRPCTLSKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load grpc tls key pair: %w", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(&tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		})),
		grpc.UnaryInterceptor(UnaryInterceptor(logger, cfg.ServiceName)),
	)

	authv1.RegisterAuthServiceServer(grpcServer, authService)

	return &Server{
		grpcServer: grpcServer,
		listener:   listener,
	}, nil
}

func (s *Server) Start() error {
	return s.grpcServer.Serve(s.listener)
}

func (s *Server) GracefulStop() {
	s.grpcServer.GracefulStop()
}

func (s *Server) Stop() {
	s.grpcServer.Stop()
}
