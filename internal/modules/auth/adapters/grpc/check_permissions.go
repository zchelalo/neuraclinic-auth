package grpc

import (
	"context"

	authv1 "github.com/zchelalo/neuraclinic-auth/gen/go/auth/v1"
)

func (s *Service) CheckPermissions(ctx context.Context, req *authv1.CheckPermissionsRequest) (*authv1.CheckPermissionsResponse, error) {
	allowed, err := s.app.CheckPermissions(ctx, req.GetAccessToken(), req.GetRequiredPermissionsKeys())
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.CheckPermissionsResponse{Allowed: allowed}, nil
}
