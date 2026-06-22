package application

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrForbidden          = errors.New("forbidden")
	ErrNotFound           = errors.New("not found")
	ErrInvalidResetCode   = errors.New("invalid reset code")
	ErrResetExpired       = errors.New("password reset expired")
	ErrTooManyAttempts    = errors.New("too many password reset attempts")
	ErrInvalidInput       = errors.New("invalid input")
)
