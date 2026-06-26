package grpcserver

import (
	"context"
	"testing"

	"github.com/zchelalo/neuraclinic-auth/internal/shared/appctx"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/i18n"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryInterceptorStoresNormalizedLanguage(t *testing.T) {
	t.Parallel()

	interceptor := UnaryInterceptor(zap.NewNop(), "auth")
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("accept-language", "es-MX,en;q=0.8"))

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/auth.v1.AuthService/VerifyToken"}, func(ctx context.Context, _ any) (any, error) {
		if got := appctx.Language(ctx); got != i18n.Spanish {
			t.Fatalf("appctx.Language() = %q, want %q", got, i18n.Spanish)
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
}
