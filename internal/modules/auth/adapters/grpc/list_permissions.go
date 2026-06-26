package grpc

import (
	"context"

	authv1 "github.com/zchelalo/neuraclinic-auth/gen/go/auth/v1"
)

func (s *Service) ListPermissions(ctx context.Context, _ *authv1.ListPermissionsRequest) (*authv1.ListPermissionsResponse, error) {
	permissions, err := s.app.ListPermissions(ctx)
	if err != nil {
		return nil, mapError(ctx, err)
	}

	resp := &authv1.ListPermissionsResponse{
		Permissions: make([]*authv1.Permission, 0, len(permissions)),
	}
	for _, permission := range permissions {
		resp.Permissions = append(resp.Permissions, permissionToProto(ctx, permission))
	}
	return resp, nil
}
