package customer

import (
	"context"
	"testing"

	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	pool := database.NewTestDB(t)
	return NewStore(pool)
}

func TestStore_Create_FindByEmail(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	c, err := s.Create(ctx, "a@b.com", "hash", "Ali", "555")
	require.NoError(t, err)
	require.NotZero(t, c.ID)

	got, err := s.FindByEmail(ctx, "a@b.com")
	require.NoError(t, err)
	require.Equal(t, "Ali", got.Name)
}

func TestStore_Create_EmailCakismasiConflict(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	_, err := s.Create(ctx, "dup@b.com", "h", "A", "1")
	require.NoError(t, err)
	_, err = s.Create(ctx, "dup@b.com", "h", "B", "2")
	require.ErrorIs(t, err, errorsx.ErrConflict)
}

func TestStore_FindByEmail_YokNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.FindByEmail(context.Background(), "yok@b.com")
	require.ErrorIs(t, err, errorsx.ErrNotFound)
}
