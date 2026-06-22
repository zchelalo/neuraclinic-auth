package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	authdb "github.com/zchelalo/neuraclinic-auth/internal/db/sqlc/auth"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	pgutil "github.com/zchelalo/neuraclinic-auth/internal/shared/postgresutil"
)

type PasswordResetRepository struct {
	q *authdb.Queries
}

func NewPasswordResetRepository(db authdb.DBTX) *PasswordResetRepository {
	return &PasswordResetRepository{q: authdb.New(db)}
}

func (r *PasswordResetRepository) Create(ctx context.Context, request ports.PasswordResetRequest) error {
	return r.q.CreatePasswordResetRequest(ctx, authdb.CreatePasswordResetRequestParams{
		ID:                  pgutil.UUID(request.ID),
		UserID:              pgutil.UUID(request.UserID),
		Email:               request.Email,
		OtpHash:             request.OTPHash,
		ResetTokenHash:      pgutil.OptionalText(request.ResetTokenHash),
		Attempts:            int32(request.Attempts),
		ExpiresAt:           pgutil.Timestamptz(request.ExpiresAt),
		ResetTokenExpiresAt: pgutil.OptionalTimestamptz(request.ResetTokenExpiresAt),
		UsedAt:              pgutil.OptionalTimestamptz(request.UsedAt),
		CreatedAt:           pgutil.Timestamptz(request.CreatedAt),
		UpdatedAt:           pgutil.Timestamptz(request.UpdatedAt),
	})
}

func (r *PasswordResetRepository) LatestActiveByEmail(ctx context.Context, email string, now time.Time) (ports.PasswordResetRequest, error) {
	row, err := r.q.LatestActivePasswordResetByEmail(ctx, authdb.LatestActivePasswordResetByEmailParams{
		Email:     email,
		ExpiresAt: pgutil.Timestamptz(now),
	})
	if err != nil {
		return ports.PasswordResetRequest{}, err
	}
	return passwordResetFromModel(row), nil
}

func (r *PasswordResetRepository) ByResetTokenHash(ctx context.Context, resetTokenHash string, now time.Time) (ports.PasswordResetRequest, error) {
	row, err := r.q.GetPasswordResetByResetTokenHash(ctx, authdb.GetPasswordResetByResetTokenHashParams{
		ResetTokenHash:      pgutil.OptionalText(&resetTokenHash),
		ResetTokenExpiresAt: pgutil.Timestamptz(now),
	})
	if err != nil {
		return ports.PasswordResetRequest{}, err
	}
	return passwordResetFromModel(row), nil
}

func (r *PasswordResetRepository) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	return r.q.IncrementPasswordResetAttempts(ctx, authdb.IncrementPasswordResetAttemptsParams{
		UpdatedAt: pgutil.Timestamptz(time.Now().UTC()),
		ID:        pgutil.UUID(id),
	})
}

func (r *PasswordResetRepository) SetResetToken(ctx context.Context, id uuid.UUID, resetTokenHash string, resetTokenExpiresAt time.Time) error {
	rows, err := r.q.SetPasswordResetToken(ctx, authdb.SetPasswordResetTokenParams{
		ResetTokenHash:      pgutil.OptionalText(&resetTokenHash),
		ResetTokenExpiresAt: pgutil.Timestamptz(resetTokenExpiresAt),
		UpdatedAt:           pgutil.Timestamptz(time.Now().UTC()),
		ID:                  pgutil.UUID(id),
	})
	if err != nil {
		return err
	}
	if rows != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error {
	rows, err := r.q.MarkPasswordResetUsed(ctx, authdb.MarkPasswordResetUsedParams{
		UsedAt: pgutil.Timestamptz(usedAt),
		ID:     pgutil.UUID(id),
	})
	if err != nil {
		return err
	}
	if rows != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func passwordResetFromModel(row authdb.PasswordResetRequest) ports.PasswordResetRequest {
	return ports.PasswordResetRequest{
		ID:                  pgutil.UUIDValue(row.ID),
		UserID:              pgutil.UUIDValue(row.UserID),
		Email:               row.Email,
		OTPHash:             row.OtpHash,
		ResetTokenHash:      pgutil.TextPtr(row.ResetTokenHash),
		Attempts:            int(row.Attempts),
		ExpiresAt:           row.ExpiresAt.Time,
		ResetTokenExpiresAt: pgutil.TimestamptzPtr(row.ResetTokenExpiresAt),
		UsedAt:              pgutil.TimestamptzPtr(row.UsedAt),
		CreatedAt:           row.CreatedAt.Time,
		UpdatedAt:           row.UpdatedAt.Time,
	}
}

var _ ports.PasswordResetRepository = (*PasswordResetRepository)(nil)
