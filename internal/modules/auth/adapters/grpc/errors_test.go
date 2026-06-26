package grpc

import (
	"context"
	"testing"

	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/appctx"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/i18n"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapErrorLocalizesByContextLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		language i18n.Language
		err      error
		code     codes.Code
		message  string
	}{
		{name: "spanish", language: i18n.Spanish, err: application.ErrInvalidCredentials, code: codes.Unauthenticated, message: "credenciales invalidas"},
		{name: "english", language: i18n.English, err: application.ErrInvalidResetCode, code: codes.InvalidArgument, message: "invalid reset code"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := appctx.WithLanguage(context.Background(), tt.language)
			got := mapError(ctx, tt.err)
			st, ok := status.FromError(got)
			if !ok {
				t.Fatalf("expected grpc status error, got %v", got)
			}
			if st.Code() != tt.code {
				t.Fatalf("status code = %s, want %s", st.Code(), tt.code)
			}
			if st.Message() != tt.message {
				t.Fatalf("status message = %q, want %q", st.Message(), tt.message)
			}
		})
	}
}
