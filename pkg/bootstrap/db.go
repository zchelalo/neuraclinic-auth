package bootstrap

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zchelalo/neuraclinic-auth/internal/db/connection"
)

func NewDB(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	dsn, err := cfg.DBDSN()
	if err != nil {
		return nil, fmt.Errorf("build postgres dsn: %w", err)
	}

	db, err := connection.NewPool(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}

	return db, nil
}
