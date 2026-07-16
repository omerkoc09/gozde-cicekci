package auth

import (
	"context"
	"testing"

	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-that-is-long-enough-32"

func TestService_CreateAdmin_ThenLogin(t *testing.T) {
	pool := database.NewTestDB(t)
	svc := NewService(NewStore(pool), testSecret)
	ctx := context.Background()

	err := svc.CreateAdmin(ctx, "cicekci", "gizli-sifre-123")
	require.NoError(t, err)

	token, err := svc.Login(ctx, "cicekci", "gizli-sifre-123")

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestService_Login_WrongPassword(t *testing.T) {
	pool := database.NewTestDB(t)
	svc := NewService(NewStore(pool), testSecret)
	ctx := context.Background()

	require.NoError(t, svc.CreateAdmin(ctx, "cicekci", "dogru-sifre-123"))

	_, err := svc.Login(ctx, "cicekci", "yanlis-sifre")

	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
}

func TestService_Login_UnknownUser(t *testing.T) {
	pool := database.NewTestDB(t)
	svc := NewService(NewStore(pool), testSecret)

	_, err := svc.Login(context.Background(), "yok-boyle", "sifre")

	require.ErrorIs(t, err, errorsx.ErrUnauthorized)
}

func TestService_CreateAdmin_DuplicateUsername(t *testing.T) {
	pool := database.NewTestDB(t)
	svc := NewService(NewStore(pool), testSecret)
	ctx := context.Background()

	require.NoError(t, svc.CreateAdmin(ctx, "cicekci", "sifre-123456"))
	err := svc.CreateAdmin(ctx, "cicekci", "baska-sifre-123")

	require.ErrorIs(t, err, errorsx.ErrConflict)
}

func TestService_CreateAdmin_ShortPassword(t *testing.T) {
	pool := database.NewTestDB(t)
	svc := NewService(NewStore(pool), testSecret)

	err := svc.CreateAdmin(context.Background(), "cicekci", "kisa")

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_PasswordIsHashed(t *testing.T) {
	pool := database.NewTestDB(t)
	store := NewStore(pool)
	svc := NewService(store, testSecret)
	ctx := context.Background()

	require.NoError(t, svc.CreateAdmin(ctx, "cicekci", "gizli-sifre-123"))

	user, err := store.FindByUsername(ctx, "cicekci")

	require.NoError(t, err)
	assert.NotEqual(t, "gizli-sifre-123", user.PasswordHash)
	assert.Contains(t, user.PasswordHash, "$2a$")
}
