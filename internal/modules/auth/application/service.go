package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/appctx"
	appcrypto "github.com/zchelalo/neuraclinic-auth/internal/shared/crypto"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/uuidx"
	"go.uber.org/zap"
)

type Config struct {
	RefreshTokenTTL          time.Duration
	PasswordResetOTPTTL      time.Duration
	PasswordResetTokenTTL    time.Duration
	PasswordResetMaxAttempts int
	TokenHashSecret          string
	Environment              string
}

type Service struct {
	cfg      Config
	users    ports.UserClient
	sessions ports.SessionRepository
	resets   ports.PasswordResetRepository
	perms    ports.PermissionRepository
	tokens   ports.TokenManager
	events   ports.EventPublisher
	now      func() time.Time
	newUUID  func() uuid.UUID
}

func NewService(
	cfg Config,
	users ports.UserClient,
	sessions ports.SessionRepository,
	resets ports.PasswordResetRepository,
	perms ports.PermissionRepository,
	tokens ports.TokenManager,
	events ports.EventPublisher,
) *Service {
	return &Service{
		cfg:      cfg,
		users:    users,
		sessions: sessions,
		resets:   resets,
		perms:    perms,
		tokens:   tokens,
		events:   events,
		now:      time.Now,
		newUUID:  uuidx.New,
	}
}

type SignInResult struct {
	AccessToken        string
	RefreshToken       string
	AccessTokenExpiry  time.Time
	RefreshTokenExpiry time.Time
}

func (s *Service) SignIn(ctx context.Context, email, password string) (SignInResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return SignInResult{}, ErrInvalidInput
	}

	user, err := s.users.VerifyPassword(ctx, email, password)
	if err != nil {
		appctx.Logger(ctx).Warn("user password verification failed", zap.String("email", email), zap.Error(err))
		return SignInResult{}, ErrInvalidCredentials
	}

	now := s.now().UTC()
	sessionID := s.newUUID()
	refreshJTI := s.newUUID()
	refreshExpiresAt := now.Add(s.cfg.RefreshTokenTTL)

	if err := s.sessions.Create(ctx, ports.Session{
		ID:         sessionID,
		UserID:     user.ID,
		RefreshJTI: refreshJTI,
		ExpiresAt:  refreshExpiresAt,
		CreatedAt:  now,
	}); err != nil {
		return SignInResult{}, fmt.Errorf("create session: %w", err)
	}

	accessToken, accessExpiresAt, err := s.tokens.IssueAccess(ctx, user.ID, sessionID)
	if err != nil {
		return SignInResult{}, fmt.Errorf("issue access token: %w", err)
	}
	refreshToken, refreshTokenExpiresAt, err := s.tokens.IssueRefresh(ctx, user.ID, sessionID, refreshJTI)
	if err != nil {
		return SignInResult{}, fmt.Errorf("issue refresh token: %w", err)
	}

	appctx.Logger(ctx).Info(
		"sign in succeeded",
		zap.String("user_id", user.ID.String()),
		zap.String("session_id", sessionID.String()),
		zap.String("access_token_fp", appcrypto.Fingerprint(accessToken)),
		zap.String("refresh_token_fp", appcrypto.Fingerprint(refreshToken)),
	)

	return SignInResult{
		AccessToken:        accessToken,
		RefreshToken:       refreshToken,
		AccessTokenExpiry:  accessExpiresAt,
		RefreshTokenExpiry: refreshTokenExpiresAt,
	}, nil
}

func (s *Service) SignOut(ctx context.Context, accessToken, refreshToken string) error {
	if accessToken == "" || refreshToken == "" {
		return ErrInvalidInput
	}

	accessClaims, err := s.tokens.VerifyAccess(ctx, accessToken)
	if err != nil {
		return ErrInvalidToken
	}
	refreshClaims, err := s.tokens.VerifyRefresh(ctx, refreshToken)
	if err != nil {
		return ErrInvalidToken
	}
	if accessClaims.UserID != refreshClaims.UserID || accessClaims.SessionID != refreshClaims.SessionID {
		return ErrForbidden
	}

	if err := s.sessions.Revoke(ctx, refreshClaims.SessionID, s.now().UTC()); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}

type RefreshResult struct {
	AccessToken        string
	RefreshToken       string
	AccessTokenExpiry  time.Time
	RefreshTokenExpiry time.Time
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (RefreshResult, error) {
	if refreshToken == "" {
		return RefreshResult{}, ErrInvalidInput
	}

	claims, err := s.tokens.VerifyRefresh(ctx, refreshToken)
	if err != nil {
		return RefreshResult{}, ErrInvalidToken
	}

	newRefreshJTI := s.newUUID()
	newRefreshExpiresAt := s.now().UTC().Add(s.cfg.RefreshTokenTTL)
	ok, err := s.sessions.ValidateAndRotateRefresh(ctx, claims.SessionID, claims.RefreshJTI, newRefreshJTI, newRefreshExpiresAt)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			return RefreshResult{}, ErrInvalidToken
		}
		return RefreshResult{}, fmt.Errorf("rotate refresh token: %w", err)
	}
	if !ok {
		_ = s.sessions.Revoke(ctx, claims.SessionID, s.now().UTC())
		appctx.Logger(ctx).Warn(
			"refresh token reuse detected",
			zap.String("user_id", claims.UserID.String()),
			zap.String("session_id", claims.SessionID.String()),
			zap.Bool("security_event", true),
		)
		return RefreshResult{}, ErrInvalidToken
	}

	accessToken, accessExpiresAt, err := s.tokens.IssueAccess(ctx, claims.UserID, claims.SessionID)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("issue access token: %w", err)
	}
	newRefreshToken, refreshExpiresAt, err := s.tokens.IssueRefresh(ctx, claims.UserID, claims.SessionID, newRefreshJTI)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("issue refresh token: %w", err)
	}

	return RefreshResult{
		AccessToken:        accessToken,
		RefreshToken:       newRefreshToken,
		AccessTokenExpiry:  accessExpiresAt,
		RefreshTokenExpiry: refreshExpiresAt,
	}, nil
}

type VerifyTokenResult struct {
	User           ports.UserIdentity
	PermissionKeys []string
}

func (s *Service) VerifyToken(ctx context.Context, accessToken string) (VerifyTokenResult, error) {
	if accessToken == "" {
		return VerifyTokenResult{}, ErrInvalidInput
	}

	claims, err := s.tokens.VerifyAccess(ctx, accessToken)
	if err != nil {
		return VerifyTokenResult{}, ErrInvalidToken
	}

	session, err := s.sessions.ByID(ctx, claims.SessionID)
	if err != nil {
		return VerifyTokenResult{}, ErrInvalidToken
	}
	now := s.now().UTC()
	if session.RevokedAt != nil || !session.ExpiresAt.After(now) || session.UserID != claims.UserID {
		return VerifyTokenResult{}, ErrInvalidToken
	}

	user, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return VerifyTokenResult{}, ErrInvalidToken
	}

	permissions, err := s.perms.AllowedPermissionKeys(ctx, user.ID, user.RoleKey.String())
	if err != nil {
		return VerifyTokenResult{}, fmt.Errorf("load permissions: %w", err)
	}

	return VerifyTokenResult{User: user, PermissionKeys: permissions}, nil
}

func (s *Service) CheckPermissions(ctx context.Context, accessToken string, required []string) (bool, error) {
	result, err := s.VerifyToken(ctx, accessToken)
	if err != nil {
		return false, err
	}
	if len(required) == 0 {
		return true, nil
	}

	allowed := make(map[string]struct{}, len(result.PermissionKeys))
	for _, key := range result.PermissionKeys {
		allowed[key] = struct{}{}
	}
	for _, key := range required {
		if _, ok := allowed[key]; !ok {
			return false, nil
		}
	}
	return true, nil
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil
	}

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		appctx.Logger(ctx).Info("password reset requested for unknown email", zap.String("email", email))
		return nil
	}

	otp, err := appcrypto.RandomDigits(6)
	if err != nil {
		return fmt.Errorf("generate otp: %w", err)
	}
	now := s.now().UTC()
	request := ports.PasswordResetRequest{
		ID:        s.newUUID(),
		UserID:    user.ID,
		Email:     email,
		OTPHash:   appcrypto.HMACSHA256(s.cfg.TokenHashSecret, otp),
		ExpiresAt: now.Add(s.cfg.PasswordResetOTPTTL),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.resets.Create(ctx, request); err != nil {
		return fmt.Errorf("store password reset request: %w", err)
	}

	event := ports.PasswordResetRequestedEvent{
		EventID:   s.newUUID().String(),
		UserID:    user.ID.String(),
		Email:     email,
		OTP:       otp,
		ExpiresAt: request.ExpiresAt,
		RequestID: appctx.RequestID(ctx),
		TraceID:   appctx.TraceID(ctx),
	}
	if err := s.events.PublishPasswordResetRequested(ctx, event); err != nil {
		appctx.Logger(ctx).Warn("password reset event publish failed", zap.Error(err))
	}

	if s.cfg.Environment != "production" {
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

func (s *Service) VerifyResetCode(ctx context.Context, email, otp string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || otp == "" {
		return "", ErrInvalidInput
	}

	request, err := s.resets.LatestActiveByEmail(ctx, email, s.now().UTC())
	if err != nil {
		return "", ErrInvalidResetCode
	}
	if request.Attempts >= s.cfg.PasswordResetMaxAttempts {
		return "", ErrTooManyAttempts
	}

	providedHash := appcrypto.HMACSHA256(s.cfg.TokenHashSecret, otp)
	if !appcrypto.SecureCompare(providedHash, request.OTPHash) {
		_ = s.resets.IncrementAttempts(ctx, request.ID)
		return "", ErrInvalidResetCode
	}

	resetToken, err := appcrypto.RandomToken(32)
	if err != nil {
		return "", fmt.Errorf("generate reset token: %w", err)
	}
	resetTokenHash := appcrypto.HMACSHA256(s.cfg.TokenHashSecret, resetToken)
	resetTokenExpiresAt := s.now().UTC().Add(s.cfg.PasswordResetTokenTTL)

	if err := s.resets.SetResetToken(ctx, request.ID, resetTokenHash, resetTokenExpiresAt); err != nil {
		return "", fmt.Errorf("store reset token: %w", err)
	}

	return resetToken, nil
}

func (s *Service) ResetPassword(ctx context.Context, resetToken, newPassword string) error {
	if resetToken == "" || newPassword == "" {
		return ErrInvalidInput
	}

	resetTokenHash := appcrypto.HMACSHA256(s.cfg.TokenHashSecret, resetToken)
	request, err := s.resets.ByResetTokenHash(ctx, resetTokenHash, s.now().UTC())
	if err != nil {
		return ErrInvalidToken
	}

	if err := s.users.UpdatePassword(ctx, request.UserID, newPassword); err != nil {
		return fmt.Errorf("update user password: %w", err)
	}

	now := s.now().UTC()
	if err := s.resets.MarkUsed(ctx, request.ID, now); err != nil {
		return fmt.Errorf("mark reset used: %w", err)
	}
	if err := s.sessions.RevokeByUserID(ctx, request.UserID, now); err != nil {
		return fmt.Errorf("revoke user sessions: %w", err)
	}

	return nil
}

func (s *Service) ListPermissions(ctx context.Context) ([]ports.Permission, error) {
	return s.perms.ListPermissions(ctx)
}
