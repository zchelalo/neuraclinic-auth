package resetpassword

import (
	"context"
	"fmt"
	"time"

	autherrors "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/errors"
	appshared "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/shared"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	appcrypto "github.com/zchelalo/neuraclinic-auth/internal/shared/crypto"
)

type UseCase struct {
	cfg      appshared.Config
	users    ports.UserClient
	resets   ports.PasswordResetRepository
	sessions ports.SessionRepository
	now      func() time.Time
}

func New(cfg appshared.Config, users ports.UserClient, resets ports.PasswordResetRepository, sessions ports.SessionRepository, runtime appshared.Runtime) *UseCase {
	runtime = runtime.Normalize()
	return &UseCase{cfg: cfg, users: users, resets: resets, sessions: sessions, now: runtime.Now}
}

type Command struct {
	ResetToken  string
	NewPassword string
}

func (uc *UseCase) Execute(ctx context.Context, cmd Command) error {
	if cmd.ResetToken == "" || cmd.NewPassword == "" {
		return autherrors.ErrInvalidInput
	}

	resetTokenHash := appcrypto.HMACSHA256(uc.cfg.TokenHashSecret, cmd.ResetToken)
	request, err := uc.resets.ByResetTokenHash(ctx, resetTokenHash, uc.now().UTC())
	if err != nil {
		return autherrors.ErrInvalidToken
	}

	if err := uc.users.UpdatePassword(ctx, request.UserID, cmd.NewPassword); err != nil {
		return fmt.Errorf("update user password: %w", err)
	}

	now := uc.now().UTC()
	if err := uc.resets.MarkUsed(ctx, request.ID, now); err != nil {
		return fmt.Errorf("mark reset used: %w", err)
	}
	if err := uc.sessions.RevokeByUserID(ctx, request.UserID, now); err != nil {
		return fmt.Errorf("revoke user sessions: %w", err)
	}

	return nil
}
