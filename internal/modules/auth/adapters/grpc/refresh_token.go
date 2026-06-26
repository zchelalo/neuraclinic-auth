package grpc

import (
	"context"

	authv1 "github.com/zchelalo/neuraclinic-auth/gen/go/auth/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	result, err := s.app.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, mapError(ctx, err)
	}

	return &authv1.RefreshTokenResponse{
		AccessToken:        result.AccessToken,
		RefreshToken:       &result.RefreshToken,
		AccessTokenExpiry:  timestamppb.New(result.AccessTokenExpiry),
		RefreshTokenExpiry: timestamppb.New(result.RefreshTokenExpiry),
	}, nil
}
