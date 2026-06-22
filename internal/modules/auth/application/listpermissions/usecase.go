package listpermissions

import (
	"context"

	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
)

type UseCase struct {
	perms ports.PermissionRepository
}

func New(perms ports.PermissionRepository) *UseCase {
	return &UseCase{perms: perms}
}

func (uc *UseCase) Execute(ctx context.Context) ([]ports.Permission, error) {
	return uc.perms.ListPermissions(ctx)
}
