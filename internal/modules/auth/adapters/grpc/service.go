package grpc

import (
	"context"
	"errors"

	authv1 "github.com/zchelalo/neuraclinic-auth/gen/go/auth/v1"
	sharedv1 "github.com/zchelalo/neuraclinic-auth/gen/go/shared/v1"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	authv1.UnimplementedAuthServiceServer
	app *application.Service
}

func NewService(app *application.Service) *Service {
	return &Service{app: app}
}

func (s *Service) SignIn(ctx context.Context, req *authv1.SignInRequest) (*authv1.SignInResponse, error) {
	result, err := s.app.SignIn(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.SignInResponse{
		AccessToken:        result.AccessToken,
		RefreshToken:       result.RefreshToken,
		AccessTokenExpiry:  timestamppb.New(result.AccessTokenExpiry),
		RefreshTokenExpiry: timestamppb.New(result.RefreshTokenExpiry),
	}, nil
}

func (s *Service) SignOut(ctx context.Context, req *authv1.SignOutRequest) (*authv1.SignOutResponse, error) {
	if err := s.app.SignOut(ctx, req.GetAccessToken(), req.GetRefreshToken()); err != nil {
		return nil, mapError(err)
	}

	return &authv1.SignOutResponse{
		Operation: operation("signed out"),
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	result, err := s.app.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.RefreshTokenResponse{
		AccessToken:        result.AccessToken,
		RefreshToken:       &result.RefreshToken,
		AccessTokenExpiry:  timestamppb.New(result.AccessTokenExpiry),
		RefreshTokenExpiry: timestamppb.New(result.RefreshTokenExpiry),
	}, nil
}

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

func (s *Service) CheckPermissions(ctx context.Context, req *authv1.CheckPermissionsRequest) (*authv1.CheckPermissionsResponse, error) {
	allowed, err := s.app.CheckPermissions(ctx, req.GetAccessToken(), req.GetRequiredPermissionsKeys())
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.CheckPermissionsResponse{Allowed: allowed}, nil
}

func (s *Service) ListPermissions(ctx context.Context, _ *authv1.ListPermissionsRequest) (*authv1.ListPermissionsResponse, error) {
	permissions, err := s.app.ListPermissions(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	resp := &authv1.ListPermissionsResponse{
		Permissions: make([]*authv1.Permission, 0, len(permissions)),
	}
	for _, permission := range permissions {
		resp.Permissions = append(resp.Permissions, permissionToProto(permission))
	}
	return resp, nil
}

func permissionToProto(permission ports.Permission) *authv1.Permission {
	key := sharedv1.PermissionKey_PERMISSION_KEY_UNSPECIFIED
	if value, ok := sharedv1.PermissionKey_value[permission.Key]; ok {
		key = sharedv1.PermissionKey(value)
	}

	resp := &authv1.Permission{
		Id:          permission.ID.String(),
		Key:         key,
		Description: permission.Description,
		CreatedAt:   timestamppb.New(permission.CreatedAt),
		UpdatedAt:   timestamppb.New(permission.UpdatedAt),
	}
	if permission.DeletedAt != nil {
		resp.DeletedAt = timestamppb.New(*permission.DeletedAt)
	}
	return resp
}

func operation(message string) *sharedv1.OperationResponse {
	return &sharedv1.OperationResponse{Message: message}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, application.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, "invalid credentials")
	case errors.Is(err, application.ErrInvalidToken):
		return status.Error(codes.Unauthenticated, "invalid token")
	case errors.Is(err, application.ErrForbidden):
		return status.Error(codes.PermissionDenied, "forbidden")
	case errors.Is(err, application.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, application.ErrInvalidResetCode):
		return status.Error(codes.InvalidArgument, "invalid reset code")
	case errors.Is(err, application.ErrResetExpired):
		return status.Error(codes.FailedPrecondition, "password reset expired")
	case errors.Is(err, application.ErrTooManyAttempts):
		return status.Error(codes.ResourceExhausted, "too many password reset attempts")
	case errors.Is(err, application.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, "invalid input")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
