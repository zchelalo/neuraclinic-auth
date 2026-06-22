package signout

import (
	"context"
	"fmt"
	"time"

	autherrors "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/errors"
	appshared "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/shared"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
)

type UseCase struct {
	sessions ports.SessionRepository
	tokens   ports.TokenManager
	now      func() time.Time
}

func New(sessions ports.SessionRepository, tokens ports.TokenManager, runtime appshared.Runtime) *UseCase {
	runtime = runtime.Normalize()
	return &UseCase{sessions: sessions, tokens: tokens, now: runtime.Now}
}

type Command struct {
	AccessToken  string
	RefreshToken string
}

func (uc *UseCase) Execute(ctx context.Context, cmd Command) error {
	if cmd.AccessToken == "" || cmd.RefreshToken == "" {
		return autherrors.ErrInvalidInput
	}

	accessClaims, err := uc.tokens.VerifyAccess(ctx, cmd.AccessToken)
	if err != nil {
		return autherrors.ErrInvalidToken
	}
	refreshClaims, err := uc.tokens.VerifyRefresh(ctx, cmd.RefreshToken)
	if err != nil {
		return autherrors.ErrInvalidToken
	}
	if accessClaims.UserID != refreshClaims.UserID || accessClaims.SessionID != refreshClaims.SessionID {
		return autherrors.ErrForbidden
	}

	if err := uc.sessions.Revoke(ctx, refreshClaims.SessionID, uc.now().UTC()); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}
