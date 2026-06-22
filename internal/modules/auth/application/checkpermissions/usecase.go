package checkpermissions

import (
	"context"

	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/verifytoken"
)

type UseCase struct {
	verifyToken *verifytoken.UseCase
}

func New(verifyToken *verifytoken.UseCase) *UseCase {
	return &UseCase{verifyToken: verifyToken}
}

type Command struct {
	AccessToken            string
	RequiredPermissionKeys []string
}

func (uc *UseCase) Execute(ctx context.Context, cmd Command) (bool, error) {
	result, err := uc.verifyToken.Execute(ctx, verifytoken.Command{AccessToken: cmd.AccessToken})
	if err != nil {
		return false, err
	}
	if len(cmd.RequiredPermissionKeys) == 0 {
		return true, nil
	}

	allowed := make(map[string]struct{}, len(result.PermissionKeys))
	for _, key := range result.PermissionKeys {
		allowed[key] = struct{}{}
	}
	for _, key := range cmd.RequiredPermissionKeys {
		if _, ok := allowed[key]; !ok {
			return false, nil
		}
	}
	return true, nil
}
