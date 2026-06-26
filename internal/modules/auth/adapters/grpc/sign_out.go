package grpc

import (
	"context"

	authv1 "github.com/zchelalo/neuraclinic-auth/gen/go/auth/v1"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/i18n"
)

func (s *Service) SignOut(ctx context.Context, req *authv1.SignOutRequest) (*authv1.SignOutResponse, error) {
	if err := s.app.SignOut(ctx, req.GetAccessToken(), req.GetRefreshToken()); err != nil {
		return nil, mapError(ctx, err)
	}

	return &authv1.SignOutResponse{
		Operation: operation(ctx, i18n.KeySignedOut),
	}, nil
}
