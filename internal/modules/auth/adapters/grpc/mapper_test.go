package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/appctx"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/i18n"
)

func TestPermissionToProtoLocalizesDescription(t *testing.T) {
	t.Parallel()

	ctx := appctx.WithLanguage(context.Background(), i18n.Spanish)
	permission := ports.Permission{
		ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Key:         "PERMISSION_KEY_USER_VIEW",
		Description: "View users",
		CreatedAt:   time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC),
	}

	got := permissionToProto(ctx, permission)
	if got.GetDescription() != "Ver usuarios" {
		t.Fatalf("permission description = %q, want %q", got.GetDescription(), "Ver usuarios")
	}
}
