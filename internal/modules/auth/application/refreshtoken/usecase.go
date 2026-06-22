package refreshtoken

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	autherrors "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/errors"
	appshared "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/shared"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/appctx"
	"go.uber.org/zap"
)

type UseCase struct {
	refreshTokenTTL time.Duration
	sessions        ports.SessionRepository
	tokens          ports.TokenManager
	now             func() time.Time
	newUUID         func() uuid.UUID
}

func New(cfg appshared.Config, sessions ports.SessionRepository, tokens ports.TokenManager, runtime appshared.Runtime) *UseCase {
	runtime = runtime.Normalize()
	return &UseCase{
		refreshTokenTTL: cfg.RefreshTokenTTL,
		sessions:        sessions,
		tokens:          tokens,
		now:             runtime.Now,
		newUUID:         runtime.NewUUID,
	}
}

type Command struct {
	RefreshToken string
}

type Result struct {
	AccessToken        string
	RefreshToken       string
	AccessTokenExpiry  time.Time
	RefreshTokenExpiry time.Time
}

func (uc *UseCase) Execute(ctx context.Context, cmd Command) (Result, error) {
	if cmd.RefreshToken == "" {
		return Result{}, autherrors.ErrInvalidInput
	}

	claims, err := uc.tokens.VerifyRefresh(ctx, cmd.RefreshToken)
	if err != nil {
		return Result{}, autherrors.ErrInvalidToken
	}

	newRefreshJTI := uc.newUUID()
	newRefreshExpiresAt := uc.now().UTC().Add(uc.refreshTokenTTL)
	ok, err := uc.sessions.ValidateAndRotateRefresh(ctx, claims.SessionID, claims.RefreshJTI, newRefreshJTI, newRefreshExpiresAt)
	if err != nil {
		if errors.Is(err, autherrors.ErrInvalidToken) {
			return Result{}, autherrors.ErrInvalidToken
		}
		return Result{}, fmt.Errorf("rotate refresh token: %w", err)
	}
	if !ok {
		_ = uc.sessions.Revoke(ctx, claims.SessionID, uc.now().UTC())
		appctx.Logger(ctx).Warn(
			"refresh token reuse detected",
			zap.String("user_id", claims.UserID.String()),
			zap.String("session_id", claims.SessionID.String()),
			zap.Bool("security_event", true),
		)
		return Result{}, autherrors.ErrInvalidToken
	}

	accessToken, accessExpiresAt, err := uc.tokens.IssueAccess(ctx, claims.UserID, claims.SessionID)
	if err != nil {
		return Result{}, fmt.Errorf("issue access token: %w", err)
	}
	newRefreshToken, refreshExpiresAt, err := uc.tokens.IssueRefresh(ctx, claims.UserID, claims.SessionID, newRefreshJTI)
	if err != nil {
		return Result{}, fmt.Errorf("issue refresh token: %w", err)
	}

	return Result{
		AccessToken:        accessToken,
		RefreshToken:       newRefreshToken,
		AccessTokenExpiry:  accessExpiresAt,
		RefreshTokenExpiry: refreshExpiresAt,
	}, nil
}
