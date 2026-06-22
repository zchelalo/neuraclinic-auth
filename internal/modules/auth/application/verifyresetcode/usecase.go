package verifyresetcode

import (
	"context"
	"fmt"
	"strings"
	"time"

	autherrors "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/errors"
	appshared "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/shared"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	appcrypto "github.com/zchelalo/neuraclinic-auth/internal/shared/crypto"
)

type UseCase struct {
	cfg    appshared.Config
	resets ports.PasswordResetRepository
	now    func() time.Time
}

func New(cfg appshared.Config, resets ports.PasswordResetRepository, runtime appshared.Runtime) *UseCase {
	runtime = runtime.Normalize()
	return &UseCase{cfg: cfg, resets: resets, now: runtime.Now}
}

type Command struct {
	Email string
	OTP   string
}

func (uc *UseCase) Execute(ctx context.Context, cmd Command) (string, error) {
	email := strings.TrimSpace(strings.ToLower(cmd.Email))
	if email == "" || cmd.OTP == "" {
		return "", autherrors.ErrInvalidInput
	}

	request, err := uc.resets.LatestActiveByEmail(ctx, email, uc.now().UTC())
	if err != nil {
		return "", autherrors.ErrInvalidResetCode
	}
	if request.Attempts >= uc.cfg.PasswordResetMaxAttempts {
		return "", autherrors.ErrTooManyAttempts
	}

	providedHash := appcrypto.HMACSHA256(uc.cfg.TokenHashSecret, cmd.OTP)
	if !appcrypto.SecureCompare(providedHash, request.OTPHash) {
		_ = uc.resets.IncrementAttempts(ctx, request.ID)
		return "", autherrors.ErrInvalidResetCode
	}

	resetToken, err := appcrypto.RandomToken(32)
	if err != nil {
		return "", fmt.Errorf("generate reset token: %w", err)
	}
	resetTokenHash := appcrypto.HMACSHA256(uc.cfg.TokenHashSecret, resetToken)
	resetTokenExpiresAt := uc.now().UTC().Add(uc.cfg.PasswordResetTokenTTL)

	if err := uc.resets.SetResetToken(ctx, request.ID, resetTokenHash, resetTokenExpiresAt); err != nil {
		return "", fmt.Errorf("store reset token: %w", err)
	}

	return resetToken, nil
}
