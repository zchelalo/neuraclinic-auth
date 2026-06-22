package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
)

type SessionRepository struct {
	db *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session ports.Session) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO auth_sessions (id, user_id, refresh_jti, revoked_at, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
	`, session.ID, session.UserID, session.RefreshJTI, session.RevokedAt, session.ExpiresAt, session.CreatedAt)
	return err
}

func (r *SessionRepository) ByID(ctx context.Context, id uuid.UUID) (ports.Session, error) {
	var session ports.Session
	var revokedAt pgtype.Timestamptz

	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, refresh_jti, revoked_at, expires_at, created_at
		FROM auth_sessions
		WHERE id = $1
	`, id).Scan(&session.ID, &session.UserID, &session.RefreshJTI, &revokedAt, &session.ExpiresAt, &session.CreatedAt)
	if err != nil {
		return ports.Session{}, err
	}
	session.RevokedAt = timestamptzPtr(revokedAt)
	return session, nil
}

func (r *SessionRepository) ValidateAndRotateRefresh(ctx context.Context, sessionID, expectedRefreshJTI, newRefreshJTI uuid.UUID, newExpiresAt time.Time) (bool, error) {
	now := time.Now().UTC()
	tag, err := r.db.Exec(ctx, `
		UPDATE auth_sessions
		SET refresh_jti = $1,
		    expires_at = $2,
		    updated_at = $3
		WHERE id = $4
		  AND refresh_jti = $5
		  AND revoked_at IS NULL
		  AND expires_at > $3
	`, newRefreshJTI, newExpiresAt, now, sessionID, expectedRefreshJTI)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *SessionRepository) Revoke(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = COALESCE(revoked_at, $1),
		    updated_at = $1
		WHERE id = $2
	`, revokedAt, sessionID)
	return err
}

func (r *SessionRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = COALESCE(revoked_at, $1),
		    updated_at = $1
		WHERE user_id = $2
		  AND revoked_at IS NULL
	`, revokedAt, userID)
	return err
}

type PasswordResetRepository struct {
	db *pgxpool.Pool
}

func NewPasswordResetRepository(db *pgxpool.Pool) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

func (r *PasswordResetRepository) Create(ctx context.Context, request ports.PasswordResetRequest) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO password_reset_requests (
			id,
			user_id,
			email,
			otp_hash,
			reset_token_hash,
			attempts,
			expires_at,
			reset_token_expires_at,
			used_at,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, request.ID, request.UserID, request.Email, request.OTPHash, request.ResetTokenHash, request.Attempts, request.ExpiresAt, request.ResetTokenExpiresAt, request.UsedAt, request.CreatedAt, request.UpdatedAt)
	return err
}

func (r *PasswordResetRepository) LatestActiveByEmail(ctx context.Context, email string, now time.Time) (ports.PasswordResetRequest, error) {
	return r.passwordResetQueryRow(ctx, `
		SELECT id, user_id, email, otp_hash, reset_token_hash, attempts, expires_at, reset_token_expires_at, used_at, created_at, updated_at
		FROM password_reset_requests
		WHERE email = $1
		  AND used_at IS NULL
		  AND expires_at > $2
		ORDER BY created_at DESC
		LIMIT 1
	`, email, now)
}

func (r *PasswordResetRepository) ByResetTokenHash(ctx context.Context, resetTokenHash string, now time.Time) (ports.PasswordResetRequest, error) {
	return r.passwordResetQueryRow(ctx, `
		SELECT id, user_id, email, otp_hash, reset_token_hash, attempts, expires_at, reset_token_expires_at, used_at, created_at, updated_at
		FROM password_reset_requests
		WHERE reset_token_hash = $1
		  AND used_at IS NULL
		  AND reset_token_expires_at > $2
		LIMIT 1
	`, resetTokenHash, now)
}

func (r *PasswordResetRepository) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE password_reset_requests
		SET attempts = attempts + 1,
		    updated_at = $1
		WHERE id = $2
	`, time.Now().UTC(), id)
	return err
}

func (r *PasswordResetRepository) SetResetToken(ctx context.Context, id uuid.UUID, resetTokenHash string, resetTokenExpiresAt time.Time) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE password_reset_requests
		SET reset_token_hash = $1,
		    reset_token_expires_at = $2,
		    updated_at = $3
		WHERE id = $4
		  AND used_at IS NULL
	`, resetTokenHash, resetTokenExpiresAt, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE password_reset_requests
		SET used_at = $1,
		    updated_at = $1
		WHERE id = $2
		  AND used_at IS NULL
	`, usedAt, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PasswordResetRepository) passwordResetQueryRow(ctx context.Context, query string, args ...any) (ports.PasswordResetRequest, error) {
	var request ports.PasswordResetRequest
	var resetTokenHash pgtype.Text
	var resetTokenExpiresAt pgtype.Timestamptz
	var usedAt pgtype.Timestamptz

	err := r.db.QueryRow(ctx, query, args...).Scan(
		&request.ID,
		&request.UserID,
		&request.Email,
		&request.OTPHash,
		&resetTokenHash,
		&request.Attempts,
		&request.ExpiresAt,
		&resetTokenExpiresAt,
		&usedAt,
		&request.CreatedAt,
		&request.UpdatedAt,
	)
	if err != nil {
		return ports.PasswordResetRequest{}, err
	}

	if resetTokenHash.Valid {
		value := resetTokenHash.String
		request.ResetTokenHash = &value
	}
	request.ResetTokenExpiresAt = timestamptzPtr(resetTokenExpiresAt)
	request.UsedAt = timestamptzPtr(usedAt)

	return request, nil
}

type PermissionRepository struct {
	db *pgxpool.Pool
}

func NewPermissionRepository(db *pgxpool.Pool) *PermissionRepository {
	return &PermissionRepository{db: db}
}

func (r *PermissionRepository) ListPermissions(ctx context.Context) ([]ports.Permission, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, key, description, created_at, updated_at, deleted_at
		FROM permissions
		WHERE deleted_at IS NULL
		ORDER BY key ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permissions := make([]ports.Permission, 0)
	for rows.Next() {
		var permission ports.Permission
		var deletedAt pgtype.Timestamptz
		if err := rows.Scan(&permission.ID, &permission.Key, &permission.Description, &permission.CreatedAt, &permission.UpdatedAt, &deletedAt); err != nil {
			return nil, err
		}
		permission.DeletedAt = timestamptzPtr(deletedAt)
		permissions = append(permissions, permission)
	}
	return permissions, rows.Err()
}

func (r *PermissionRepository) AllowedPermissionKeys(ctx context.Context, userID uuid.UUID, roleKey string) ([]string, error) {
	allowed := make(map[string]bool)

	rows, err := r.db.Query(ctx, `
		SELECT permission_key, effect
		FROM role_permissions
		WHERE role_key = $1
	`, roleKey)
	if err != nil {
		return nil, err
	}
	if err := applyPermissionRows(rows, allowed); err != nil {
		return nil, err
	}

	rows, err = r.db.Query(ctx, `
		SELECT permission_key, effect
		FROM user_permission_overrides
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	if err := applyPermissionRows(rows, allowed); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(allowed))
	for key, ok := range allowed {
		if ok {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func applyPermissionRows(rows pgx.Rows, allowed map[string]bool) error {
	defer rows.Close()

	for rows.Next() {
		var key string
		var effect string
		if err := rows.Scan(&key, &effect); err != nil {
			return err
		}
		switch effect {
		case "allow":
			allowed[key] = true
		case "deny":
			allowed[key] = false
		default:
			return fmt.Errorf("unknown permission effect: %s", effect)
		}
	}
	return rows.Err()
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

var _ ports.SessionRepository = (*SessionRepository)(nil)
var _ ports.PasswordResetRepository = (*PasswordResetRepository)(nil)
var _ ports.PermissionRepository = (*PermissionRepository)(nil)
