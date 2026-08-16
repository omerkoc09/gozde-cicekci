package category

import (
	"testing"

	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uc kategori aynı eksende üç kategori oluşturur, oluşturulma sırasıyla döner.
func ucKategori(t *testing.T, svc *Service, axis Axis) []int64 {
	t.Helper()
	ids := make([]int64, 0, 3)
	for _, ad := range []string{"Bir", "İki", "Üç"} {
		c, err := svc.Create(t.Context(), CreateInput{Name: ad, Axis: axis, IsActive: true})
		require.NoError(t, err)
		ids = append(ids, c.ID)
	}
	return ids
}

// Yeni kategori sona eklenir — panel artık sıra sayısı sormuyor, sunucu
// karar veriyor.
func TestService_Create_SonaEkler(t *testing.T) {
	svc, ctx := newTestService(t)

	ids := ucKategori(t, svc, AxisOccasion)

	list, err := svc.ListAdmin(ctx)
	require.NoError(t, err)

	sira := make(map[int64]int, len(list))
	for _, c := range list {
		sira[c.ID] = c.SortOrder
	}
	assert.Less(t, sira[ids[0]], sira[ids[1]])
	assert.Less(t, sira[ids[1]], sira[ids[2]])
}

// İki eksen bağımsız sıralanır: type eksenindeki ilk kategori, occasion
// eksenindekiler kaç tane olursa olsun kendi ekseninin başında olmalı.
func TestService_Create_EksenlerBagimsizSiralanir(t *testing.T) {
	svc, ctx := newTestService(t)

	ucKategori(t, svc, AxisOccasion)

	ilkType, err := svc.Create(ctx, CreateInput{Name: "Gül", Axis: AxisType, IsActive: true})
	require.NoError(t, err)

	assert.Equal(t, 0, ilkType.SortOrder)
}

func TestService_Reorder_VerilenSirayaGoreYenidenNumaralar(t *testing.T) {
	svc, ctx := newTestService(t)

	ids := ucKategori(t, svc, AxisOccasion)

	// Sonuncuyu başa al.
	yeni := []int64{ids[2], ids[0], ids[1]}
	require.NoError(t, svc.Reorder(ctx, AxisOccasion, yeni))

	list, err := svc.ListAdmin(ctx)
	require.NoError(t, err)

	sira := make(map[int64]int, len(list))
	for _, c := range list {
		sira[c.ID] = c.SortOrder
	}
	assert.Equal(t, 0, sira[ids[2]])
	assert.Equal(t, 1, sira[ids[0]])
	assert.Equal(t, 2, sira[ids[1]])
}

// Eksik ID reddedilir. Aksi halde listede olmayan kategori 0'da kalır ve
// sıralama sessizce bozulur.
func TestService_Reorder_EksikIDReddedilir(t *testing.T) {
	svc, ctx := newTestService(t)

	ids := ucKategori(t, svc, AxisOccasion)

	err := svc.Reorder(ctx, AxisOccasion, []int64{ids[0], ids[1]})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

// Başka eksenin kategorisi karıştırılamaz.
func TestService_Reorder_YabanciEksenReddedilir(t *testing.T) {
	svc, ctx := newTestService(t)

	ids := ucKategori(t, svc, AxisOccasion)

	yabanci, err := svc.Create(ctx, CreateInput{Name: "Gül", Axis: AxisType, IsActive: true})
	require.NoError(t, err)

	err = svc.Reorder(ctx, AxisOccasion, []int64{ids[0], ids[1], ids[2], yabanci.ID})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Reorder_TekrarEdenIDReddedilir(t *testing.T) {
	svc, ctx := newTestService(t)

	ids := ucKategori(t, svc, AxisOccasion)

	err := svc.Reorder(ctx, AxisOccasion, []int64{ids[0], ids[0], ids[1]})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}
