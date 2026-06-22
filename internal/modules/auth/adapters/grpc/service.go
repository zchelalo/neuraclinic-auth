package grpc

import (
	authv1 "github.com/zchelalo/neuraclinic-auth/gen/go/auth/v1"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/application"
)

type Service struct {
	authv1.UnimplementedAuthServiceServer
	app *application.Service
}

func NewService(app *application.Service) *Service {
	return &Service{app: app}
}
