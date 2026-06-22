package application

import autherrors "github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application/errors"

var ErrInvalidCredentials = autherrors.ErrInvalidCredentials
var ErrInvalidToken = autherrors.ErrInvalidToken
var ErrForbidden = autherrors.ErrForbidden
var ErrNotFound = autherrors.ErrNotFound
var ErrInvalidResetCode = autherrors.ErrInvalidResetCode
var ErrResetExpired = autherrors.ErrResetExpired
var ErrTooManyAttempts = autherrors.ErrTooManyAttempts
var ErrInvalidInput = autherrors.ErrInvalidInput
