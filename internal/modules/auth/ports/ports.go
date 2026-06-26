package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	sharedv1 "github.com/zchelalo/neuraclinic-auth/gen/go/shared/v1"
)

type UserIdentity struct {
	ID             uuid.UUID
	Email          string
	RoleKey        sharedv1.RoleKey
	PsychologistID *uuid.UUID
	AdminID        *uuid.UUID
}

type Session struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	RefreshJTI uuid.UUID
	RevokedAt  *time.Time
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

type PasswordResetRequest struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	Email               string
	OTPHash             string
	ResetTokenHash      *string
	Attempts            int
	ExpiresAt           time.Time
	ResetTokenExpiresAt *time.Time
	UsedAt              *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type Permission struct {
	ID          uuid.UUID
	Key         string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

type UserClient interface {
	VerifyPassword(ctx context.Context, email, password string) (UserIdentity, error)
	FindByID(ctx context.Context, id uuid.UUID) (UserIdentity, error)
	FindByEmail(ctx context.Context, email string) (UserIdentity, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, newPassword string) error
	Close() error
}

type SessionRepository interface {
	Create(ctx context.Context, session Session) error
	ByID(ctx context.Context, id uuid.UUID) (Session, error)
	ValidateAndRotateRefresh(ctx context.Context, sessionID, expectedRefreshJTI, newRefreshJTI uuid.UUID, newExpiresAt time.Time) (bool, error)
	Revoke(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error
	RevokeByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error
}

type PasswordResetRepository interface {
	Create(ctx context.Context, request PasswordResetRequest) error
	LatestActiveByEmail(ctx context.Context, email string, now time.Time) (PasswordResetRequest, error)
	ByResetTokenHash(ctx context.Context, resetTokenHash string, now time.Time) (PasswordResetRequest, error)
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
	SetResetToken(ctx context.Context, id uuid.UUID, resetTokenHash string, resetTokenExpiresAt time.Time) error
	MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error
}

type PermissionRepository interface {
	ListPermissions(ctx context.Context) ([]Permission, error)
	AllowedPermissionKeys(ctx context.Context, userID uuid.UUID, roleKey string) ([]string, error)
}

type EventPublisher interface {
	PublishPasswordResetRequested(ctx context.Context, event PasswordResetRequestedEvent) error
	Close() error
}

type AccessClaims struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	ExpiresAt time.Time
}

type RefreshClaims struct {
	UserID     uuid.UUID
	SessionID  uuid.UUID
	RefreshJTI uuid.UUID
	ExpiresAt  time.Time
}

type TokenManager interface {
	IssueAccess(ctx context.Context, userID, sessionID uuid.UUID) (string, time.Time, error)
	IssueRefresh(ctx context.Context, userID, sessionID, refreshJTI uuid.UUID) (string, time.Time, error)
	VerifyAccess(ctx context.Context, token string) (AccessClaims, error)
	VerifyRefresh(ctx context.Context, token string) (RefreshClaims, error)
}

type PasswordResetRequestedEvent struct {
	EventID   string    `json:"event_id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	OTP       string    `json:"otp"`
	Language  string    `json:"language"`
	ExpiresAt time.Time `json:"expires_at"`
	RequestID string    `json:"request_id"`
	TraceID   string    `json:"trace_id"`
}
