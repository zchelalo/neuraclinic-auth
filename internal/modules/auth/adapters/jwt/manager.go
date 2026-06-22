package jwt

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
)

type KeyPaths struct {
	AccessPrivatePath  string
	AccessPublicPath   string
	RefreshPrivatePath string
	RefreshPublicPath  string
}

type Manager struct {
	accessPrivate  *rsa.PrivateKey
	accessPublic   *rsa.PublicKey
	refreshPrivate *rsa.PrivateKey
	refreshPublic  *rsa.PublicKey
	issuer         string
	accessTTL      time.Duration
	refreshTTL     time.Duration
	now            func() time.Time
}

func NewManager(paths KeyPaths, issuer string, accessTTL, refreshTTL time.Duration) (*Manager, error) {
	accessPrivate, err := loadPrivateKey(paths.AccessPrivatePath)
	if err != nil {
		return nil, fmt.Errorf("load access private key: %w", err)
	}
	accessPublic, err := loadPublicKey(paths.AccessPublicPath)
	if err != nil {
		return nil, fmt.Errorf("load access public key: %w", err)
	}
	refreshPrivate, err := loadPrivateKey(paths.RefreshPrivatePath)
	if err != nil {
		return nil, fmt.Errorf("load refresh private key: %w", err)
	}
	refreshPublic, err := loadPublicKey(paths.RefreshPublicPath)
	if err != nil {
		return nil, fmt.Errorf("load refresh public key: %w", err)
	}

	return &Manager{
		accessPrivate:  accessPrivate,
		accessPublic:   accessPublic,
		refreshPrivate: refreshPrivate,
		refreshPublic:  refreshPublic,
		issuer:         issuer,
		accessTTL:      accessTTL,
		refreshTTL:     refreshTTL,
		now:            time.Now,
	}, nil
}

type accessClaims struct {
	SessionID string `json:"sid"`
	jwtlib.RegisteredClaims
}

type refreshClaims struct {
	SessionID string `json:"sid"`
	jwtlib.RegisteredClaims
}

func (m *Manager) IssueAccess(_ context.Context, userID, sessionID uuid.UUID) (string, time.Time, error) {
	now := m.now().UTC()
	expiresAt := now.Add(m.accessTTL)
	claims := accessClaims{
		SessionID: sessionID.String(),
		RegisteredClaims: jwtlib.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(expiresAt),
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	signed, err := token.SignedString(m.accessPrivate)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func (m *Manager) IssueRefresh(_ context.Context, userID, sessionID, refreshJTI uuid.UUID) (string, time.Time, error) {
	now := m.now().UTC()
	expiresAt := now.Add(m.refreshTTL)
	claims := refreshClaims{
		SessionID: sessionID.String(),
		RegisteredClaims: jwtlib.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID.String(),
			ID:        refreshJTI.String(),
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(expiresAt),
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	signed, err := token.SignedString(m.refreshPrivate)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func (m *Manager) VerifyAccess(_ context.Context, token string) (ports.AccessClaims, error) {
	parsed, err := jwtlib.ParseWithClaims(token, &accessClaims{}, func(t *jwtlib.Token) (any, error) {
		if t.Method.Alg() != jwtlib.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.accessPublic, nil
	}, jwtlib.WithIssuer(m.issuer), jwtlib.WithValidMethods([]string{jwtlib.SigningMethodRS256.Alg()}))
	if err != nil {
		return ports.AccessClaims{}, err
	}

	claims, ok := parsed.Claims.(*accessClaims)
	if !ok || !parsed.Valid {
		return ports.AccessClaims{}, fmt.Errorf("invalid access token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return ports.AccessClaims{}, err
	}
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return ports.AccessClaims{}, err
	}

	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	return ports.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: expiresAt}, nil
}

func (m *Manager) VerifyRefresh(_ context.Context, token string) (ports.RefreshClaims, error) {
	parsed, err := jwtlib.ParseWithClaims(token, &refreshClaims{}, func(t *jwtlib.Token) (any, error) {
		if t.Method.Alg() != jwtlib.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.refreshPublic, nil
	}, jwtlib.WithIssuer(m.issuer), jwtlib.WithValidMethods([]string{jwtlib.SigningMethodRS256.Alg()}))
	if err != nil {
		return ports.RefreshClaims{}, err
	}

	claims, ok := parsed.Claims.(*refreshClaims)
	if !ok || !parsed.Valid {
		return ports.RefreshClaims{}, fmt.Errorf("invalid refresh token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return ports.RefreshClaims{}, err
	}
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return ports.RefreshClaims{}, err
	}
	refreshJTI, err := uuid.Parse(claims.ID)
	if err != nil {
		return ports.RefreshClaims{}, err
	}

	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	return ports.RefreshClaims{UserID: userID, SessionID: sessionID, RefreshJTI: refreshJTI, ExpiresAt: expiresAt}, nil
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("invalid pem")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an rsa private key")
	}
	return rsaKey, nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("invalid pem")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an rsa public key")
	}
	return rsaKey, nil
}
