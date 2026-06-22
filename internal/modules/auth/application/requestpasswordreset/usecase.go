package requestpasswordreset

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	appshared "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/shared"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/appctx"
	appcrypto "github.com/zchelalo/neuraclinic-auth/internal/shared/crypto"
	"go.uber.org/zap"
)

type UseCase struct {
	cfg     appshared.Config
	users   ports.UserClient
	resets  ports.PasswordResetRepository
	events  ports.EventPublisher
	now     func() time.Time
	newUUID func() uuid.UUID
}

func New(cfg appshared.Config, users ports.UserClient, resets ports.PasswordResetRepository, events ports.EventPublisher, runtime appshared.Runtime) *UseCase {
	runtime = runtime.Normalize()
	return &UseCase{cfg: cfg, users: users, resets: resets, events: events, now: runtime.Now, newUUID: runtime.NewUUID}
}

type Command struct {
	Email string
}

func (uc *UseCase) Execute(ctx context.Context, cmd Command) error {
	email := strings.TrimSpace(strings.ToLower(cmd.Email))
	if email == "" {
		return nil
	}

	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		appctx.Logger(ctx).Info("password reset requested for unknown email", zap.String("email", email))
		return nil
	}

	otp, err := appcrypto.RandomDigits(6)
	if err != nil {
		return fmt.Errorf("generate otp: %w", err)
	}
	now := uc.now().UTC()
	request := ports.PasswordResetRequest{
		ID:        uc.newUUID(),
		UserID:    user.ID,
		Email:     email,
		OTPHash:   appcrypto.HMACSHA256(uc.cfg.TokenHashSecret, otp),
		ExpiresAt: now.Add(uc.cfg.PasswordResetOTPTTL),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.resets.Create(ctx, request); err != nil {
		return fmt.Errorf("store password reset request: %w", err)
	}

	event := ports.PasswordResetRequestedEvent{
		EventID:   uc.newUUID().String(),
		UserID:    user.ID.String(),
		Email:     email,
		OTP:       otp,
		ExpiresAt: request.ExpiresAt,
		RequestID: appctx.RequestID(ctx),
		TraceID:   appctx.TraceID(ctx),
	}
	if err := uc.events.PublishPasswordResetRequested(ctx, event); err != nil {
		appctx.Logger(ctx).Warn("password reset event publish failed", zap.Error(err))
	}

	if uc.cfg.Environment != "production" {
		appctx.Logger(ctx).Info(
			"password reset otp generated",
			zap.String("user_id", user.ID.String()),
			zap.String("email", email),
			zap.String("otp", otp),
			zap.Time("expires_at", request.ExpiresAt),
		)
	}

	return nil
}
