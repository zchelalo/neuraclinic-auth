package shared

import (
	"time"

	"github.com/google/uuid"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/uuidx"
)

type Config struct {
	RefreshTokenTTL          time.Duration
	PasswordResetOTPTTL      time.Duration
	PasswordResetTokenTTL    time.Duration
	PasswordResetMaxAttempts int
	TokenHashSecret          string
	Environment              string
}

type Runtime struct {
	Now     func() time.Time
	NewUUID func() uuid.UUID
}

func DefaultRuntime() Runtime {
	return Runtime{
		Now:     time.Now,
		NewUUID: uuidx.New,
	}
}

func (r Runtime) Normalize() Runtime {
	if r.Now == nil {
		r.Now = time.Now
	}
	if r.NewUUID == nil {
		r.NewUUID = uuidx.New
	}
	return r
}
