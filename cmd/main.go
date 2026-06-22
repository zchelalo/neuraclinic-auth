package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zchelalo/neuraclinic-auth/internal/bootstrap"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/adapters/events"
	authgrpc "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/adapters/grpc"
	jwtadapter "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/adapters/jwt"
	authpg "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/adapters/postgres"
	userv1client "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/adapters/users"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	grpcserver "github.com/zchelalo/neuraclinic-auth/internal/server/grpc"
	"go.uber.org/zap"
)

func main() {
	cfg, err := bootstrap.LoadConfig(".env")
	if err != nil {
		panic(err)
	}

	logger, err := bootstrap.NewLogger(cfg.Environment)
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	ctx := context.Background()
	db, err := bootstrap.NewDB(ctx, cfg)
	if err != nil {
		logger.Fatal("cannot initialize db", zap.Error(err))
	}
	defer db.Close()

	tokenManager, err := jwtadapter.NewManager(jwtadapter.KeyPaths{
		AccessPrivatePath:  cfg.JWTAccessPrivateKeyPath,
		AccessPublicPath:   cfg.JWTAccessPublicKeyPath,
		RefreshPrivatePath: cfg.JWTRefreshPrivateKeyPath,
		RefreshPublicPath:  cfg.JWTRefreshPublicKeyPath,
	}, cfg.ServiceName, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		logger.Fatal("cannot initialize jwt manager", zap.Error(err))
	}

	usersClient, err := userv1client.New(userv1client.Config{
		Addr:               cfg.UsersGRPCAddr,
		TLSEnabled:         cfg.UsersGRPCTLSEnabled,
		CACertPath:         cfg.UsersGRPCCACertPath,
		InsecureSkipVerify: cfg.UsersGRPCInsecureSkipVerify,
		InternalToken:      cfg.InternalServiceToken,
	})
	if err != nil {
		logger.Fatal("cannot initialize users client", zap.Error(err))
	}
	defer func() { _ = usersClient.Close() }()

	var publisher ports.EventPublisher = events.NewNoopPublisher()
	if cfg.RabbitMQURL != "" {
		rabbitPublisher, err := events.NewRabbitPublisher(cfg.RabbitMQURL, cfg.RabbitMQExchange, cfg.RabbitMQPasswordResetRoutingKey)
		if err != nil {
			logger.Warn("rabbitmq unavailable; password reset events will be logged only", zap.Error(err))
		} else {
			publisher = rabbitPublisher
		}
	}
	defer func() { _ = publisher.Close() }()

	sessionRepo := authpg.NewSessionRepository(db)
	resetRepo := authpg.NewPasswordResetRepository(db)
	permissionRepo := authpg.NewPermissionRepository(db)

	authApp := application.NewService(
		application.Config{
			RefreshTokenTTL:          cfg.RefreshTokenTTL,
			PasswordResetOTPTTL:      cfg.PasswordResetOTPTTL,
			PasswordResetTokenTTL:    cfg.PasswordResetTokenTTL,
			PasswordResetMaxAttempts: cfg.PasswordResetMaxAttempts,
			TokenHashSecret:          cfg.TokenHashSecret,
			Environment:              cfg.Environment,
		},
		usersClient,
		sessionRepo,
		resetRepo,
		permissionRepo,
		tokenManager,
		publisher,
	)

	server, err := grpcserver.New(cfg, logger, authgrpc.NewService(authApp))
	if err != nil {
		logger.Fatal("cannot create grpc server", zap.Error(err))
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("grpc server starting", zap.Int("port", cfg.Port))
		errCh <- server.Start()
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigs:
		logger.Info("signal received, shutting down", zap.String("signal", sig.String()))
	case err := <-errCh:
		logger.Error("grpc server stopped", zap.Error(err))
	}

	stopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		logger.Warn("graceful shutdown timed out; forcing stop")
		server.Stop()
	}

	logger.Info("shutdown complete")
}
