package productoption

import (
	"context"
	"testing"

	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	return NewStore(database.NewTestDB(t)), context.Background()
}

func TestStore_CreateGroup_DegerleriBosBaslar(t *testing.T) {
	store, ctx := newTestStore(t)

	g, err := store.CreateGroup(ctx, CreateGroupInput{
		Name: "Ambalaj Rengi", Kind: KindColor,
	}, "ambalaj-rengi")

	require.NoError(t, err)
	assert.Equal(t, "Ambalaj Rengi", g.Name)
	assert.Equal(t, "ambalaj-rengi", g.Slug)
	assert.Equal(t, KindColor, g.Kind)
	assert.True(t, g.IsActive)
	assert.Empty(t, g.Values)
}

func TestStore_ListGroups_DegerleriDeGetirir(t *testing.T) {
	store, ctx := newTestStore(t)

	g, err := store.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor}, "ambalaj")
	require.NoError(t, err)

	_, err = store.CreateValue(ctx, CreateValueInput{
		GroupID: g.ID, Name: "Pembe", SwatchHex: "#F0A6CA",
	})
	require.NoError(t, err)

	list, err := store.ListGroups(ctx, false)

	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Len(t, list[0].Values, 1)
	assert.Equal(t, "Pembe", list[0].Values[0].Name)
	assert.Equal(t, "#F0A6CA", list[0].Values[0].SwatchHex)
}

// onlyActive=true public akış için: pasif grup DA pasif değer DE gelmemeli.
func TestStore_ListGroups_OnlyActive_PasifleriEler(t *testing.T) {
	store, ctx := newTestStore(t)

	aktif, err := store.CreateGroup(ctx, CreateGroupInput{Name: "Aktif", Kind: KindColor}, "aktif")
	require.NoError(t, err)
	pasifGrup, err := store.CreateGroup(ctx, CreateGroupInput{Name: "Pasif", Kind: KindColor}, "pasif")
	require.NoError(t, err)

	yok := false
	_, err = store.UpdateGroup(ctx, pasifGrup.ID, UpdateGroupInput{IsActive: &yok})
	require.NoError(t, err)

	_, err = store.CreateValue(ctx, CreateValueInput{GroupID: aktif.ID, Name: "Pembe", SwatchHex: "#F0A6CA"})
	require.NoError(t, err)
	pasifDeger, err := store.CreateValue(ctx, CreateValueInput{GroupID: aktif.ID, Name: "Eski", SwatchHex: "#000000"})
	require.NoError(t, err)
	_, err = store.UpdateValue(ctx, pasifDeger.ID, UpdateValueInput{IsActive: &yok})
	require.NoError(t, err)

	list, err := store.ListGroups(ctx, true)

	require.NoError(t, err)
	require.Len(t, list, 1, "pasif grup gelmemeli")
	require.Len(t, list[0].Values, 1, "pasif değer gelmemeli")
	assert.Equal(t, "Pembe", list[0].Values[0].Name)
}

func TestStore_DeleteGroup_DegerleriDeSiler(t *testing.T) {
	store, ctx := newTestStore(t)

	g, err := store.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor}, "ambalaj")
	require.NoError(t, err)
	_, err = store.CreateValue(ctx, CreateValueInput{GroupID: g.ID, Name: "Pembe", SwatchHex: "#F0A6CA"})
	require.NoError(t, err)

	require.NoError(t, store.DeleteGroup(ctx, g.ID))

	list, err := store.ListGroups(ctx, false)
	require.NoError(t, err)
	assert.Empty(t, list)
}
