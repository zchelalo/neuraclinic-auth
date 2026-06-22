package bootstrap

import (
	"context"
	"fmt"

	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/adapters/events"
	authgrpc "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/adapters/grpc"
	jwtadapter "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/adapters/jwt"
	authpg "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/adapters/persistence/postgres"
	userv1client "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/adapters/users"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	grpcserver "github.com/zchelalo/neuraclinic-auth/internal/server/grpc"
	"go.uber.org/zap"
)

type App struct {
	Server  *grpcserver.Server
	Cleanup func(context.Context) error
}

func InitApp(ctx context.Context, logger *zap.Logger, cfg Config) (*App, error) {
	db, err := NewDB(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize db: %w", err)
	}

	tokenManager, err := jwtadapter.NewManager(jwtadapter.KeyPaths{
		AccessPrivatePath:  cfg.JWTAccessPrivateKeyPath,
		AccessPublicPath:   cfg.JWTAccessPublicKeyPath,
		RefreshPrivatePath: cfg.JWTRefreshPrivateKeyPath,
		RefreshPublicPath:  cfg.JWTRefreshPublicKeyPath,
	}, cfg.ServiceName, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot initialize jwt manager: %w", err)
	}

	usersClient, err := userv1client.New(userv1client.Config{
		Addr:               cfg.UsersGRPCAddr,
		TLSEnabled:         cfg.UsersGRPCTLSEnabled,
		CACertPath:         cfg.UsersGRPCCACertPath,
		InsecureSkipVerify: cfg.UsersGRPCInsecureSkipVerify,
		InternalToken:      cfg.InternalServiceToken,
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot initialize users client: %w", err)
	}

	var publisher ports.EventPublisher = events.NewNoopPublisher()
	if cfg.RabbitMQURL != "" {
		rabbitPublisher, err := events.NewRabbitPublisher(cfg.RabbitMQURL, cfg.RabbitMQExchange, cfg.RabbitMQPasswordResetRoutingKey)
		if err != nil {
			logger.Warn("rabbitmq unavailable; password reset events will be logged only", zap.Error(err))
		} else {
			publisher = rabbitPublisher
		}
	}

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

	server, err := grpcserver.New(grpcserver.Config{
		Port:            cfg.Port,
		ServiceName:     cfg.ServiceName,
		TLSCertFilePath: cfg.GRPCTLSCertPath,
		TLSKeyFilePath:  cfg.GRPCTLSKeyPath,
	}, logger, authgrpc.NewService(authApp))
	if err != nil {
		db.Close()
		_ = usersClient.Close()
		_ = publisher.Close()
		return nil, fmt.Errorf("cannot create grpc server: %w", err)
	}

	return &App{
		Server: server,
		Cleanup: func(context.Context) error {
			server.GracefulStop()
			_ = usersClient.Close()
			_ = publisher.Close()
			db.Close()
			return nil
		},
	}, nil
}
