package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	sharedv1 "github.com/zchelalo/neuraclinic-auth/gen/go/shared/v1"
	"github.com/zchelalo/neuraclinic-auth/internal/modules/auth/ports"
)

func TestSignInCreatesSessionAndTokens(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	sessionID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	refreshJTI := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

	sessions := &fakeSessions{}
	service := newTestService(&fakeUsers{identity: testIdentity(userID)}, sessions, &fakeResets{}, &fakePerms{}, &fakeTokens{}, &fakeEvents{})
	service.now = func() time.Time { return now }
	service.newUUID = uuidSeq(sessionID, refreshJTI)

	result, err := service.SignIn(ctx, "USER@EXAMPLE.COM", "Password123!")
	if err != nil {
		t.Fatalf("SignIn returned error: %v", err)
	}

	if result.AccessToken != "access-token" || result.RefreshToken != "refresh-token" {
		t.Fatalf("unexpected tokens: %#v", result)
	}
	if len(sessions.created) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions.created))
	}
	if sessions.created[0].UserID != userID || sessions.created[0].ID != sessionID || sessions.created[0].RefreshJTI != refreshJTI {
		t.Fatalf("unexpected created session: %#v", sessions.created[0])
	}
}

func TestRefreshTokenRevokesSessionOnReuse(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	sessionID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	refreshJTI := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	newRefreshJTI := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	sessions := &fakeSessions{rotateOK: false}
	tokens := &fakeTokens{refreshClaims: ports.RefreshClaims{UserID: userID, SessionID: sessionID, RefreshJTI: refreshJTI}}
	service := newTestService(&fakeUsers{}, sessions, &fakeResets{}, &fakePerms{}, tokens, &fakeEvents{})
	service.newUUID = uuidSeq(newRefreshJTI)

	_, err := service.RefreshToken(ctx, "refresh-token")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
	if sessions.revokedSessionID != sessionID {
		t.Fatalf("expected session %s to be revoked, got %s", sessionID, sessions.revokedSessionID)
	}
}

func TestPasswordResetFlow(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	resetID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

	users := &fakeUsers{identity: testIdentity(userID)}
	sessions := &fakeSessions{}
	resets := &fakeResets{}
	events := &fakeEvents{}
	service := newTestService(users, sessions, resets, &fakePerms{}, &fakeTokens{}, events)
	service.now = func() time.Time { return now }
	service.newUUID = uuidSeq(resetID, eventID)

	if err := service.RequestPasswordReset(ctx, "user@example.com"); err != nil {
		t.Fatalf("RequestPasswordReset returned error: %v", err)
	}
	if events.last.OTP == "" {
		t.Fatal("expected reset OTP event")
	}

	resetToken, err := service.VerifyResetCode(ctx, "user@example.com", events.last.OTP)
	if err != nil {
		t.Fatalf("VerifyResetCode returned error: %v", err)
	}
	if resetToken == "" {
		t.Fatal("expected reset token")
	}

	if err := service.ResetPassword(ctx, resetToken, "NewPassword123!"); err != nil {
		t.Fatalf("ResetPassword returned error: %v", err)
	}
	if users.updatedPassword != "NewPassword123!" {
		t.Fatalf("password was not updated")
	}
	if resets.usedID != resetID {
		t.Fatalf("expected reset request marked used")
	}
	if sessions.revokedUserID != userID {
		t.Fatalf("expected user sessions revoked")
	}
}

func newTestService(users ports.UserClient, sessions ports.SessionRepository, resets ports.PasswordResetRepository, perms ports.PermissionRepository, tokens ports.TokenManager, events ports.EventPublisher) *Service {
	return NewService(Config{
		RefreshTokenTTL:          time.Hour,
		PasswordResetOTPTTL:      10 * time.Minute,
		PasswordResetTokenTTL:    15 * time.Minute,
		PasswordResetMaxAttempts: 5,
		TokenHashSecret:          "test-secret",
		Environment:              "test",
	}, users, sessions, resets, perms, tokens, events)
}

func testIdentity(id uuid.UUID) ports.UserIdentity {
	adminID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	return ports.UserIdentity{
		ID:      id,
		Email:   "user@example.com",
		RoleKey: sharedv1.RoleKey_ROLE_KEY_ADMIN,
		AdminID: &adminID,
	}
}

func uuidSeq(values ...uuid.UUID) func() uuid.UUID {
	index := 0
	return func() uuid.UUID {
		if index >= len(values) {
			return uuid.Must(uuid.NewV7())
		}
		value := values[index]
		index++
		return value
	}
}

type fakeUsers struct {
	identity        ports.UserIdentity
	updatedPassword string
}

func (f *fakeUsers) VerifyPassword(context.Context, string, string) (ports.UserIdentity, error) {
	return f.identity, nil
}

func (f *fakeUsers) FindByID(context.Context, uuid.UUID) (ports.UserIdentity, error) {
	return f.identity, nil
}

func (f *fakeUsers) FindByEmail(context.Context, string) (ports.UserIdentity, error) {
	return f.identity, nil
}

func (f *fakeUsers) UpdatePassword(_ context.Context, _ uuid.UUID, newPassword string) error {
	f.updatedPassword = newPassword
	return nil
}

func (f *fakeUsers) Close() error {
	return nil
}

type fakeSessions struct {
	created          []ports.Session
	rotateOK         bool
	revokedSessionID uuid.UUID
	revokedUserID    uuid.UUID
}

func (f *fakeSessions) Create(_ context.Context, session ports.Session) error {
	f.created = append(f.created, session)
	return nil
}

func (f *fakeSessions) ByID(context.Context, uuid.UUID) (ports.Session, error) {
	return ports.Session{ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (f *fakeSessions) ValidateAndRotateRefresh(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time) (bool, error) {
	return f.rotateOK, nil
}

func (f *fakeSessions) Revoke(_ context.Context, sessionID uuid.UUID, _ time.Time) error {
	f.revokedSessionID = sessionID
	return nil
}

func (f *fakeSessions) RevokeByUserID(_ context.Context, userID uuid.UUID, _ time.Time) error {
	f.revokedUserID = userID
	return nil
}

type fakeResets struct {
	request ports.PasswordResetRequest
	usedID  uuid.UUID
}

func (f *fakeResets) Create(_ context.Context, request ports.PasswordResetRequest) error {
	f.request = request
	return nil
}

func (f *fakeResets) LatestActiveByEmail(context.Context, string, time.Time) (ports.PasswordResetRequest, error) {
	return f.request, nil
}

func (f *fakeResets) ByResetTokenHash(_ context.Context, resetTokenHash string, _ time.Time) (ports.PasswordResetRequest, error) {
	if f.request.ResetTokenHash == nil || *f.request.ResetTokenHash != resetTokenHash {
		return ports.PasswordResetRequest{}, ErrInvalidToken
	}
	return f.request, nil
}

func (f *fakeResets) IncrementAttempts(context.Context, uuid.UUID) error {
	f.request.Attempts++
	return nil
}

func (f *fakeResets) SetResetToken(_ context.Context, _ uuid.UUID, resetTokenHash string, resetTokenExpiresAt time.Time) error {
	f.request.ResetTokenHash = &resetTokenHash
	f.request.ResetTokenExpiresAt = &resetTokenExpiresAt
	return nil
}

func (f *fakeResets) MarkUsed(_ context.Context, id uuid.UUID, usedAt time.Time) error {
	f.usedID = id
	f.request.UsedAt = &usedAt
	return nil
}

type fakePerms struct{}

func (f *fakePerms) ListPermissions(context.Context) ([]ports.Permission, error) {
	return nil, nil
}

func (f *fakePerms) AllowedPermissionKeys(context.Context, uuid.UUID, string) ([]string, error) {
	return []string{"PERMISSION_KEY_USER_VIEW"}, nil
}

type fakeTokens struct {
	refreshClaims ports.RefreshClaims
}

func (f *fakeTokens) IssueAccess(context.Context, uuid.UUID, uuid.UUID) (string, time.Time, error) {
	return "access-token", time.Now().Add(time.Minute), nil
}

func (f *fakeTokens) IssueRefresh(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (string, time.Time, error) {
	return "refresh-token", time.Now().Add(time.Hour), nil
}

func (f *fakeTokens) VerifyAccess(context.Context, string) (ports.AccessClaims, error) {
	return ports.AccessClaims{}, nil
}

func (f *fakeTokens) VerifyRefresh(context.Context, string) (ports.RefreshClaims, error) {
	return f.refreshClaims, nil
}

type fakeEvents struct {
	last ports.PasswordResetRequestedEvent
}

func (f *fakeEvents) PublishPasswordResetRequested(_ context.Context, event ports.PasswordResetRequestedEvent) error {
	f.last = event
	return nil
}

func (f *fakeEvents) Close() error {
	return nil
}
