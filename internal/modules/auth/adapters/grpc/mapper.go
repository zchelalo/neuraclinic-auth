package grpc

import (
	authv1 "github.com/zchelalo/neuraclinic-auth/gen/go/auth/v1"
	sharedv1 "github.com/zchelalo/neuraclinic-auth/gen/go/shared/v1"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func permissionToProto(permission ports.Permission) *authv1.Permission {
	key := sharedv1.PermissionKey_PERMISSION_KEY_UNSPECIFIED
	if value, ok := sharedv1.PermissionKey_value[permission.Key]; ok {
		key = sharedv1.PermissionKey(value)
	}

	resp := &authv1.Permission{
		Id:          permission.ID.String(),
		Key:         key,
		Description: permission.Description,
		CreatedAt:   timestamppb.New(permission.CreatedAt),
		UpdatedAt:   timestamppb.New(permission.UpdatedAt),
	}
	if permission.DeletedAt != nil {
		resp.DeletedAt = timestamppb.New(*permission.DeletedAt)
	}
	return resp
}

func operation(message string) *sharedv1.OperationResponse {
	return &sharedv1.OperationResponse{Message: message}
}
