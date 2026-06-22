package grpc

import (
	"errors"

	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
