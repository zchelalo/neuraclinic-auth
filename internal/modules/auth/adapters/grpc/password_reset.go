package grpc

import (
	"context"

	authv1 "github.com/zchelalo/neuraclinic-auth/gen/go/auth/v1"
)

func (s *Service) RequestPasswordReset(ctx context.Context, req *authv1.RequestPasswordResetRequest) (*authv1.RequestPasswordResetResponse, error) {
	if err := s.app.RequestPasswordReset(ctx, req.GetEmail()); err != nil {
		return nil, mapError(err)
	}

	return &authv1.RequestPasswordResetResponse{
		Operation: operation("password reset requested"),
	}, nil
}

func (s *Service) VerifyResetCode(ctx context.Context, req *authv1.VerifyResetCodeRequest) (*authv1.VerifyResetCodeResponse, error) {
	resetToken, err := s.app.VerifyResetCode(ctx, req.GetEmail(), req.GetOtp())
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.VerifyResetCodeResponse{ResetToken: resetToken}, nil
}

func (s *Service) ResetPassword(ctx context.Context, req *authv1.ResetPasswordRequest) (*authv1.ResetPasswordResponse, error) {
	if err := s.app.ResetPassword(ctx, req.GetResetToken(), req.GetNewPassword()); err != nil {
		return nil, mapError(err)
	}

	return &authv1.ResetPasswordResponse{
		Operation: operation("password reset completed"),
	}, nil
}
