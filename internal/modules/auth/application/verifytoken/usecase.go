package verifytoken

import (
	"context"
	"fmt"
	"time"

	autherrors "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/errors"
	appshared "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/shared"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
)

type UseCase struct {
	users    ports.UserClient
	sessions ports.SessionRepository
	perms    ports.PermissionRepository
	tokens   ports.TokenManager
	now      func() time.Time
}

func New(users ports.UserClient, sessions ports.SessionRepository, perms ports.PermissionRepository, tokens ports.TokenManager, runtime appshared.Runtime) *UseCase {
	runtime = runtime.Normalize()
	return &UseCase{users: users, sessions: sessions, perms: perms, tokens: tokens, now: runtime.Now}
}

type Command struct {
	AccessToken string
}

type Result struct {
	User           ports.UserIdentity
	PermissionKeys []string
}

func (uc *UseCase) Execute(ctx context.Context, cmd Command) (Result, error) {
	if cmd.AccessToken == "" {
		return Result{}, autherrors.ErrInvalidInput
	}

	claims, err := uc.tokens.VerifyAccess(ctx, cmd.AccessToken)
	if err != nil {
		return Result{}, autherrors.ErrInvalidToken
	}

	session, err := uc.sessions.ByID(ctx, claims.SessionID)
	if err != nil {
		return Result{}, autherrors.ErrInvalidToken
	}
	now := uc.now().UTC()
	if session.RevokedAt != nil || !session.ExpiresAt.After(now) || session.UserID != claims.UserID {
		return Result{}, autherrors.ErrInvalidToken
	}

	user, err := uc.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return Result{}, autherrors.ErrInvalidToken
	}

	permissions, err := uc.perms.AllowedPermissionKeys(ctx, user.ID, user.RoleKey.String())
	if err != nil {
		return Result{}, fmt.Errorf("load permissions: %w", err)
	}

	return Result{User: user, PermissionKeys: permissions}, nil
}
