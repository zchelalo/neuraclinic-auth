package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	authdb "github.com/zchelalo/neuraclinic-auth/internal/db/sqlc/auth"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	pgutil "github.com/zchelalo/neuraclinic-auth/internal/shared/postgresutil"
)

type PermissionRepository struct {
	q *authdb.Queries
}

func NewPermissionRepository(db authdb.DBTX) *PermissionRepository {
	return &PermissionRepository{q: authdb.New(db)}
}

func (r *PermissionRepository) ListPermissions(ctx context.Context) ([]ports.Permission, error) {
	rows, err := r.q.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}

	permissions := make([]ports.Permission, 0, len(rows))
	for _, row := range rows {
		permissions = append(permissions, ports.Permission{
			ID:          pgutil.UUIDValue(row.ID),
			Key:         row.Key,
			Description: row.Description,
			CreatedAt:   row.CreatedAt.Time,
			UpdatedAt:   row.UpdatedAt.Time,
			DeletedAt:   pgutil.TimestamptzPtr(row.DeletedAt),
		})
	}
	return permissions, nil
}

func (r *PermissionRepository) AllowedPermissionKeys(ctx context.Context, userID uuid.UUID, roleKey string) ([]string, error) {
	allowed := make(map[string]bool)

	roleRows, err := r.q.ListRolePermissionEffects(ctx, roleKey)
	if err != nil {
		return nil, err
	}
	for _, row := range roleRows {
		if err := applyPermissionEffect(allowed, row.PermissionKey, row.Effect); err != nil {
			return nil, err
		}
	}

	overrideRows, err := r.q.ListUserPermissionOverrideEffects(ctx, pgutil.UUID(userID))
	if err != nil {
		return nil, err
	}
	for _, row := range overrideRows {
		if err := applyPermissionEffect(allowed, row.PermissionKey, row.Effect); err != nil {
			return nil, err
		}
	}

	keys := make([]string, 0, len(allowed))
	for key, ok := range allowed {
		if ok {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func applyPermissionEffect(allowed map[string]bool, key, effect string) error {
	switch effect {
	case "allow":
		allowed[key] = true
	case "deny":
		allowed[key] = false
	default:
		return fmt.Errorf("unknown permission effect: %s", effect)
	}
	return nil
}

var _ ports.PermissionRepository = (*PermissionRepository)(nil)
