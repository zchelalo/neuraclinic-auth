package grpc

import (
	"context"

	authv1 "github.com/zchelalo/neuraclinic-auth/gen/go/auth/v1"
)

func (s *Service) VerifyToken(ctx context.Context, req *authv1.VerifyTokenRequest) (*authv1.VerifyTokenResponse, error) {
	result, err := s.app.VerifyToken(ctx, req.GetAccessToken())
	if err != nil {
		return nil, mapError(err)
	}

	resp := &authv1.VerifyTokenResponse{
		UserId:          result.User.ID.String(),
		RoleKey:         result.User.RoleKey,
		PermissionsKeys: result.PermissionKeys,
	}
	if result.User.PsychologistID != nil {
		value := result.User.PsychologistID.String()
		resp.PsychologistId = &value
	}
	if result.User.AdminID != nil {
		value := result.User.AdminID.String()
		resp.AdminId = &value
	}
	return resp, nil
}
