package customer

import (
	"context"
	"fmt"
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

func TestStore_List_AramaEmailVeyaName(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Create(ctx, "ayse@example.com", "h", "Ayşe Yılmaz", "5551")
	require.NoError(t, err)
	_, err = s.Create(ctx, "mehmet@example.com", "h", "Mehmet Kaya", "5552")
	require.NoError(t, err)
	_, err = s.Create(ctx, "fatma@other.com", "h", "Fatma Şahin", "5553")
	require.NoError(t, err)

	// name üzerinden arama
	byName, err := s.List(ctx, "Ayşe", 10, 0)
	require.NoError(t, err)
	require.Len(t, byName, 1)
	require.Equal(t, "ayse@example.com", byName[0].Email)

	// email üzerinden arama (domain ortak)
	byEmail, err := s.List(ctx, "example.com", 10, 0)
	require.NoError(t, err)
	require.Len(t, byEmail, 2)

	// q boşsa hepsi
	all, err := s.List(ctx, "", 10, 0)
	require.NoError(t, err)
	require.Len(t, all, 3)

	// eşleşmeyen arama
	none, err := s.List(ctx, "yokbukelime", 10, 0)
	require.NoError(t, err)
	require.Empty(t, none)

	// PasswordHash listeye sızmamalı
	require.Empty(t, all[0].PasswordHash)
}

func TestStore_List_Sayfalama(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := s.Create(ctx, fmt.Sprintf("u%d@b.com", i), "h", fmt.Sprintf("User %d", i), "555")
		require.NoError(t, err)
	}

	page1, err := s.List(ctx, "", 2, 0)
	require.NoError(t, err)
	require.Len(t, page1, 2)

	page2, err := s.List(ctx, "", 2, 2)
	require.NoError(t, err)
	require.Len(t, page2, 2)

	// sayfalar arası çakışma olmamalı (created_at DESC sıralı, farklı kayıtlar)
	require.NotEqual(t, page1[0].ID, page2[0].ID)
	require.NotEqual(t, page1[1].ID, page2[0].ID)

	total, err := s.Count(ctx, "")
	require.NoError(t, err)
	require.Equal(t, 5, total)

	totalFiltered, err := s.Count(ctx, "User 1")
	require.NoError(t, err)
	require.Equal(t, 1, totalFiltered)
}
