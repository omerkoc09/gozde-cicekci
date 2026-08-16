package productoption

import (
	"context"
	"testing"

	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*Service, context.Context) {
	t.Helper()
	return NewService(NewStore(database.NewTestDB(t))), context.Background()
}

func TestService_CreateGroup_SlugUretir(t *testing.T) {
	svc, ctx := newTestService(t)

	g, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj Rengi", Kind: KindColor})

	require.NoError(t, err)
	assert.Equal(t, "ambalaj-rengi", g.Slug)
}

func TestService_CreateGroup_AyniAdSlugCakismasi(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor})
	require.NoError(t, err)

	ikinci, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindText})

	require.NoError(t, err)
	assert.Equal(t, "ambalaj-2", ikinci.Slug)
}

func TestService_CreateGroup_GecersizKindReddedilir(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: "renk"})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_CreateGroup_BosAdReddedilir(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "   ", Kind: KindColor})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

// Renk grubunda hex zorunlu ve geçerli formatta olmalı; aksi halde
// müşteri sayfasında görünmez bir nokta çıkar.
func TestService_CreateValue_RenkGrubundaHexZorunlu(t *testing.T) {
	svc, ctx := newTestService(t)

	g, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor})
	require.NoError(t, err)

	_, err = svc.CreateValue(ctx, CreateValueInput{GroupID: g.ID, Name: "Pembe"})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_CreateValue_GecersizHexReddedilir(t *testing.T) {
	svc, ctx := newTestService(t)

	g, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Ambalaj", Kind: KindColor})
	require.NoError(t, err)

	for _, hex := range []string{"F0A6CA", "#XYZ", "#F0A6C", "pembe"} {
		_, err = svc.CreateValue(ctx, CreateValueInput{GroupID: g.ID, Name: "Pembe", SwatchHex: hex})
		assert.ErrorIs(t, err, errorsx.ErrInvalidInput, "hex %q reddedilmeliydi", hex)
	}
}

// Metin grubunda hex gönderilse bile yok sayılır — kind='text' değerinde
// hex saklamak anlamsız.
func TestService_CreateValue_MetinGrubundaHexTemizlenir(t *testing.T) {
	svc, ctx := newTestService(t)

	g, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Boy", Kind: KindText})
	require.NoError(t, err)

	v, err := svc.CreateValue(ctx, CreateValueInput{GroupID: g.ID, Name: "Orta", SwatchHex: "#FFFFFF"})

	require.NoError(t, err)
	assert.Empty(t, v.SwatchHex)
}

func TestService_ReorderGroups_YenidenNumaralar(t *testing.T) {
	svc, ctx := newTestService(t)

	ids := make([]int64, 0, 3)
	for _, ad := range []string{"Bir", "İki", "Üç"} {
		g, err := svc.CreateGroup(ctx, CreateGroupInput{Name: ad, Kind: KindColor})
		require.NoError(t, err)
		ids = append(ids, g.ID)
	}

	require.NoError(t, svc.ReorderGroups(ctx, []int64{ids[2], ids[0], ids[1]}))

	list, err := svc.ListGroups(ctx, false)
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, ids[2], list[0].ID)
	assert.Equal(t, ids[0], list[1].ID)
	assert.Equal(t, ids[1], list[2].ID)
}

func TestService_ReorderGroups_EksikIDReddedilir(t *testing.T) {
	svc, ctx := newTestService(t)

	g, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Bir", Kind: KindColor})
	require.NoError(t, err)
	_, err = svc.CreateGroup(ctx, CreateGroupInput{Name: "İki", Kind: KindColor})
	require.NoError(t, err)

	err = svc.ReorderGroups(ctx, []int64{g.ID})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_ReorderValues_BaskaGrubunDegeriReddedilir(t *testing.T) {
	svc, ctx := newTestService(t)

	g1, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "Bir", Kind: KindColor})
	require.NoError(t, err)
	g2, err := svc.CreateGroup(ctx, CreateGroupInput{Name: "İki", Kind: KindColor})
	require.NoError(t, err)

	v1, err := svc.CreateValue(ctx, CreateValueInput{GroupID: g1.ID, Name: "Pembe", SwatchHex: "#F0A6CA"})
	require.NoError(t, err)
	yabanci, err := svc.CreateValue(ctx, CreateValueInput{GroupID: g2.ID, Name: "Mavi", SwatchHex: "#0000FF"})
	require.NoError(t, err)

	err = svc.ReorderValues(ctx, g1.ID, []int64{v1.ID, yabanci.ID})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}
