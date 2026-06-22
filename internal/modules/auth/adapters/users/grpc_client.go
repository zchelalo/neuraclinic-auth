package users

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/google/uuid"
	sharedv1 "github.com/zchelalo/neuraclinic-auth/gen/go/shared/v1"
	userv1 "github.com/zchelalo/neuraclinic-auth/gen/go/user/v1"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	headerInternalServiceToken = "x-internal-service-token"
	headerUserID               = "x-user-id"
)

type Config struct {
	Addr               string
	TLSEnabled         bool
	CACertPath         string
	InsecureSkipVerify bool
	InternalToken      string
}

type Client struct {
	conn          *grpc.ClientConn
	client        userv1.UserServiceClient
	internalToken string
}

func New(cfg Config) (*Client, error) {
	creds, err := transportCredentials(cfg)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.NewClient(cfg.Addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("create users grpc client: %w", err)
	}

	return &Client{
		conn:          conn,
		client:        userv1.NewUserServiceClient(conn),
		internalToken: cfg.InternalToken,
	}, nil
}

func (c *Client) VerifyPassword(ctx context.Context, email, password string) (ports.UserIdentity, error) {
	ctx = c.withInternalToken(ctx)
	resp, err := c.client.VerifyPassword(ctx, &userv1.UserServiceVerifyPasswordRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return ports.UserIdentity{}, err
	}
	if !resp.GetValid() || resp.GetUser() == nil {
		return ports.UserIdentity{}, fmt.Errorf("invalid credentials")
	}
	return identityFromVerifyPassword(resp)
}

func (c *Client) FindByID(ctx context.Context, id uuid.UUID) (ports.UserIdentity, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, headerUserID, id.String())
	resp, err := c.client.FindById(ctx, &userv1.UserServiceFindByIdRequest{Id: id.String()})
	if err != nil {
		return ports.UserIdentity{}, err
	}
	return identityFromFindByID(resp)
}

func (c *Client) FindByEmail(ctx context.Context, email string) (ports.UserIdentity, error) {
	ctx = c.withInternalToken(ctx)
	resp, err := c.client.FindByEmail(ctx, &userv1.UserServiceFindByEmailRequest{Email: email})
	if err != nil {
		return ports.UserIdentity{}, err
	}
	return identityFromFindByEmail(resp)
}

func (c *Client) UpdatePassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	ctx = c.withInternalToken(ctx)
	_, err := c.client.UpdatePassword(ctx, &userv1.UserServiceUpdatePasswordRequest{
		Id:          id.String(),
		NewPassword: newPassword,
	})
	return err
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) withInternalToken(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, headerInternalServiceToken, c.internalToken)
}

func transportCredentials(cfg Config) (credentials.TransportCredentials, error) {
	if !cfg.TLSEnabled {
		return insecure.NewCredentials(), nil
	}

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if cfg.InsecureSkipVerify {
		tlsCfg.InsecureSkipVerify = true
		return credentials.NewTLS(tlsCfg), nil
	}
	if cfg.CACertPath != "" {
		ca, err := os.ReadFile(cfg.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("read users ca cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("append users ca cert")
		}
		tlsCfg.RootCAs = pool
	}

	return credentials.NewTLS(tlsCfg), nil
}

func identityFromFindByID(resp *userv1.UserServiceFindByIdResponse) (ports.UserIdentity, error) {
	return identityFromParts(resp.GetUser(), resp.GetAdmin(), resp.GetPsychologist())
}

func identityFromFindByEmail(resp *userv1.UserServiceFindByEmailResponse) (ports.UserIdentity, error) {
	return identityFromParts(resp.GetUser(), resp.GetAdmin(), resp.GetPsychologist())
}

func identityFromVerifyPassword(resp *userv1.UserServiceVerifyPasswordResponse) (ports.UserIdentity, error) {
	return identityFromParts(resp.GetUser(), resp.GetAdmin(), resp.GetPsychologist())
}

func identityFromParts(user *userv1.User, admin *userv1.AdminProfile, psychologist *userv1.PsychologistProfile) (ports.UserIdentity, error) {
	if user == nil {
		return ports.UserIdentity{}, fmt.Errorf("missing user")
	}
	userID, err := uuid.Parse(user.GetId())
	if err != nil {
		return ports.UserIdentity{}, err
	}

	identity := ports.UserIdentity{
		ID:      userID,
		Email:   user.GetEmail(),
		RoleKey: user.GetRoleKey(),
	}
	if identity.RoleKey == sharedv1.RoleKey_ROLE_KEY_UNSPECIFIED {
		return ports.UserIdentity{}, fmt.Errorf("missing role")
	}
	if admin != nil {
		adminID, err := uuid.Parse(admin.GetId())
		if err != nil {
			return ports.UserIdentity{}, err
		}
		identity.AdminID = &adminID
	}
	if psychologist != nil {
		psychologistID, err := uuid.Parse(psychologist.GetId())
		if err != nil {
			return ports.UserIdentity{}, err
		}
		identity.PsychologistID = &psychologistID
	}

	return identity, nil
}
