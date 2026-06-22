package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	authdb "github.com/zchelalo/neuraclinic-auth/internal/db/sqlc/auth"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	pgutil "github.com/zchelalo/neuraclinic-auth/internal/shared/postgresutil"
)

type SessionRepository struct {
	q *authdb.Queries
}

func NewSessionRepository(db authdb.DBTX) *SessionRepository {
	return &SessionRepository{q: authdb.New(db)}
}

func (r *SessionRepository) Create(ctx context.Context, session ports.Session) error {
	return r.q.CreateSession(ctx, authdb.CreateSessionParams{
		ID:         pgutil.UUID(session.ID),
		UserID:     pgutil.UUID(session.UserID),
		RefreshJti: pgutil.UUID(session.RefreshJTI),
		RevokedAt:  pgutil.OptionalTimestamptz(session.RevokedAt),
		ExpiresAt:  pgutil.Timestamptz(session.ExpiresAt),
		CreatedAt:  pgutil.Timestamptz(session.CreatedAt),
	})
}

func (r *SessionRepository) ByID(ctx context.Context, id uuid.UUID) (ports.Session, error) {
	row, err := r.q.GetSessionByID(ctx, pgutil.UUID(id))
	if err != nil {
		return ports.Session{}, err
	}

	return ports.Session{
		ID:         pgutil.UUIDValue(row.ID),
		UserID:     pgutil.UUIDValue(row.UserID),
		RefreshJTI: pgutil.UUIDValue(row.RefreshJti),
		RevokedAt:  pgutil.TimestamptzPtr(row.RevokedAt),
		ExpiresAt:  row.ExpiresAt.Time,
		CreatedAt:  row.CreatedAt.Time,
	}, nil
}

func (r *SessionRepository) ValidateAndRotateRefresh(ctx context.Context, sessionID, expectedRefreshJTI, newRefreshJTI uuid.UUID, newExpiresAt time.Time) (bool, error) {
	now := time.Now().UTC()
	rows, err := r.q.RotateSessionRefresh(ctx, authdb.RotateSessionRefreshParams{
		RefreshJti:   pgutil.UUID(newRefreshJTI),
		ExpiresAt:    pgutil.Timestamptz(newExpiresAt),
		UpdatedAt:    pgutil.Timestamptz(now),
		ID:           pgutil.UUID(sessionID),
		RefreshJti_2: pgutil.UUID(expectedRefreshJTI),
	})
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}

func (r *SessionRepository) Revoke(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error {
	return r.q.RevokeSession(ctx, authdb.RevokeSessionParams{
		UpdatedAt: pgutil.Timestamptz(revokedAt),
		ID:        pgutil.UUID(sessionID),
	})
}

func (r *SessionRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	return r.q.RevokeSessionsByUserID(ctx, authdb.RevokeSessionsByUserIDParams{
		UpdatedAt: pgutil.Timestamptz(revokedAt),
		UserID:    pgutil.UUID(userID),
	})
}

var _ ports.SessionRepository = (*SessionRepository)(nil)
