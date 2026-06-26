package grpc

import (
	"context"

	authv1 "github.com/zchelalo/neuraclinic-auth/gen/go/auth/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) SignIn(ctx context.Context, req *authv1.SignInRequest) (*authv1.SignInResponse, error) {
	result, err := s.app.SignIn(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, mapError(ctx, err)
	}

	return &authv1.SignInResponse{
		AccessToken:        result.AccessToken,
		RefreshToken:       result.RefreshToken,
		AccessTokenExpiry:  timestamppb.New(result.AccessTokenExpiry),
		RefreshTokenExpiry: timestamppb.New(result.RefreshTokenExpiry),
	}, nil
}
