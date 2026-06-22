package application

import (
	"context"

	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/checkpermissions"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/listpermissions"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/refreshtoken"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/requestpasswordreset"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/resetpassword"
	appshared "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/shared"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/signin"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/signout"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/verifyresetcode"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/verifytoken"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
)

type Config = appshared.Config
type Runtime = appshared.Runtime

type SignInResult = signin.Result
type RefreshResult = refreshtoken.Result
type VerifyTokenResult = verifytoken.Result

type Service struct {
	signIn               *signin.UseCase
	signOut              *signout.UseCase
	refreshToken         *refreshtoken.UseCase
	verifyToken          *verifytoken.UseCase
	checkPermissions     *checkpermissions.UseCase
	requestPasswordReset *requestpasswordreset.UseCase
	verifyResetCode      *verifyresetcode.UseCase
	resetPassword        *resetpassword.UseCase
	listPermissions      *listpermissions.UseCase
}

func NewService(
	cfg Config,
	users ports.UserClient,
	sessions ports.SessionRepository,
	resets ports.PasswordResetRepository,
	perms ports.PermissionRepository,
	tokens ports.TokenManager,
	events ports.EventPublisher,
) *Service {
	return NewServiceWithRuntime(cfg, users, sessions, resets, perms, tokens, events, appshared.DefaultRuntime())
}

func NewServiceWithRuntime(
	cfg Config,
	users ports.UserClient,
	sessions ports.SessionRepository,
	resets ports.PasswordResetRepository,
	perms ports.PermissionRepository,
	tokens ports.TokenManager,
	events ports.EventPublisher,
	runtime Runtime,
) *Service {
	verifyTokenUC := verifytoken.New(users, sessions, perms, tokens, runtime)
	return &Service{
		signIn:               signin.New(cfg, users, sessions, tokens, runtime),
		signOut:              signout.New(sessions, tokens, runtime),
		refreshToken:         refreshtoken.New(cfg, sessions, tokens, runtime),
		verifyToken:          verifyTokenUC,
		checkPermissions:     checkpermissions.New(verifyTokenUC),
		requestPasswordReset: requestpasswordreset.New(cfg, users, resets, events, runtime),
		verifyResetCode:      verifyresetcode.New(cfg, resets, runtime),
		resetPassword:        resetpassword.New(cfg, users, resets, sessions, runtime),
		listPermissions:      listpermissions.New(perms),
	}
}

func DefaultRuntime() Runtime {
	return appshared.DefaultRuntime()
}

func (s *Service) SignIn(ctx context.Context, email, password string) (SignInResult, error) {
	return s.signIn.Execute(ctx, signin.Command{Email: email, Password: password})
}

func (s *Service) SignOut(ctx context.Context, accessToken, refreshToken string) error {
	return s.signOut.Execute(ctx, signout.Command{AccessToken: accessToken, RefreshToken: refreshToken})
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (RefreshResult, error) {
	return s.refreshToken.Execute(ctx, refreshtoken.Command{RefreshToken: refreshToken})
}

func (s *Service) VerifyToken(ctx context.Context, accessToken string) (VerifyTokenResult, error) {
	return s.verifyToken.Execute(ctx, verifytoken.Command{AccessToken: accessToken})
}

func (s *Service) CheckPermissions(ctx context.Context, accessToken string, required []string) (bool, error) {
	return s.checkPermissions.Execute(ctx, checkpermissions.Command{
		AccessToken:            accessToken,
		RequiredPermissionKeys: required,
	})
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	return s.requestPasswordReset.Execute(ctx, requestpasswordreset.Command{Email: email})
}

func (s *Service) VerifyResetCode(ctx context.Context, email, otp string) (string, error) {
	return s.verifyResetCode.Execute(ctx, verifyresetcode.Command{Email: email, OTP: otp})
}

func (s *Service) ResetPassword(ctx context.Context, resetToken, newPassword string) error {
	return s.resetPassword.Execute(ctx, resetpassword.Command{ResetToken: resetToken, NewPassword: newPassword})
}

func (s *Service) ListPermissions(ctx context.Context) ([]ports.Permission, error) {
	return s.listPermissions.Execute(ctx)
}
