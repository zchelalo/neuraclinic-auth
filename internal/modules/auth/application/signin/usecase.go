package signin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	autherrors "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/errors"
	appshared "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/shared"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/appctx"
	appcrypto "github.com/zchelalo/neuraclinic-auth/internal/shared/crypto"
	"go.uber.org/zap"
)

type UseCase struct {
	refreshTokenTTL time.Duration
	users           ports.UserClient
	sessions        ports.SessionRepository
	tokens          ports.TokenManager
	now             func() time.Time
	newUUID         func() uuid.UUID
}

func New(cfg appshared.Config, users ports.UserClient, sessions ports.SessionRepository, tokens ports.TokenManager, runtime appshared.Runtime) *UseCase {
	runtime = runtime.Normalize()
	return &UseCase{
		refreshTokenTTL: cfg.RefreshTokenTTL,
		users:           users,
		sessions:        sessions,
		tokens:          tokens,
		now:             runtime.Now,
		newUUID:         runtime.NewUUID,
	}
}

type Command struct {
	Email    string
	Password string
}

type Result struct {
	AccessToken        string
	RefreshToken       string
	AccessTokenExpiry  time.Time
	RefreshTokenExpiry time.Time
}

func (uc *UseCase) Execute(ctx context.Context, cmd Command) (Result, error) {
	email := strings.TrimSpace(strings.ToLower(cmd.Email))
	if email == "" || cmd.Password == "" {
		return Result{}, autherrors.ErrInvalidInput
	}

	user, err := uc.users.VerifyPassword(ctx, email, cmd.Password)
	if err != nil {
		appctx.Logger(ctx).Warn("user password verification failed", zap.String("email", email), zap.Error(err))
		return Result{}, autherrors.ErrInvalidCredentials
	}

	now := uc.now().UTC()
	sessionID := uc.newUUID()
	refreshJTI := uc.newUUID()
	refreshExpiresAt := now.Add(uc.refreshTokenTTL)

	if err := uc.sessions.Create(ctx, ports.Session{
		ID:         sessionID,
		UserID:     user.ID,
		RefreshJTI: refreshJTI,
		ExpiresAt:  refreshExpiresAt,
		CreatedAt:  now,
	}); err != nil {
		return Result{}, fmt.Errorf("create session: %w", err)
	}

	accessToken, accessExpiresAt, err := uc.tokens.IssueAccess(ctx, user.ID, sessionID)
	if err != nil {
		return Result{}, fmt.Errorf("issue access token: %w", err)
	}
	refreshToken, refreshTokenExpiresAt, err := uc.tokens.IssueRefresh(ctx, user.ID, sessionID, refreshJTI)
	if err != nil {
		return Result{}, fmt.Errorf("issue refresh token: %w", err)
	}

	appctx.Logger(ctx).Info(
		"sign in succeeded",
		zap.String("user_id", user.ID.String()),
		zap.String("session_id", sessionID.String()),
		zap.String("access_token_fp", appcrypto.Fingerprint(accessToken)),
		zap.String("refresh_token_fp", appcrypto.Fingerprint(refreshToken)),
	)

	return Result{
		AccessToken:        accessToken,
		RefreshToken:       refreshToken,
		AccessTokenExpiry:  accessExpiresAt,
		RefreshTokenExpiry: refreshTokenExpiresAt,
	}, nil
}
