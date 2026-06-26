package grpc

import (
	"context"
	"errors"

	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/appctx"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/i18n"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapError(ctx context.Context, err error) error {
	language := appctx.Language(ctx)
	switch {
	case errors.Is(err, application.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, i18n.Message(language, i18n.KeyInvalidCredentials))
	case errors.Is(err, application.ErrInvalidToken):
		return status.Error(codes.Unauthenticated, i18n.Message(language, i18n.KeyInvalidToken))
	case errors.Is(err, application.ErrForbidden):
		return status.Error(codes.PermissionDenied, i18n.Message(language, i18n.KeyForbidden))
	case errors.Is(err, application.ErrNotFound):
		return status.Error(codes.NotFound, i18n.Message(language, i18n.KeyNotFound))
	case errors.Is(err, application.ErrInvalidResetCode):
		return status.Error(codes.InvalidArgument, i18n.Message(language, i18n.KeyInvalidResetCode))
	case errors.Is(err, application.ErrResetExpired):
		return status.Error(codes.FailedPrecondition, i18n.Message(language, i18n.KeyResetExpired))
	case errors.Is(err, application.ErrTooManyAttempts):
		return status.Error(codes.ResourceExhausted, i18n.Message(language, i18n.KeyTooManyAttempts))
	case errors.Is(err, application.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, i18n.Message(language, i18n.KeyInvalidInput))
	default:
		return status.Error(codes.Internal, i18n.Message(language, i18n.KeyInternalServerError))
	}
}
