package slider

import (
	"testing"

	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ucSlayt üç slayt oluşturur, oluşturulma sırasıyla ID'lerini döner.
func ucSlayt(t *testing.T, svc *Service) []int64 {
	t.Helper()
	ids := make([]int64, 0, 3)
	for _, baslik := range []string{"Bir", "İki", "Üç"} {
		in := validInput()
		in.Title = baslik
		s, err := svc.Create(t.Context(), in, makeJPEG(t, 2400, 1200))
		require.NoError(t, err)
		ids = append(ids, s.ID)
	}
	return ids
}

// Yeni slayt sona eklenir — panel artık sıra sayısı sormuyor.
func TestService_Create_SonaEkler(t *testing.T) {
	svc, _, ctx := newTestService(t)

	ids := ucSlayt(t, svc)

	list, err := svc.ListAdmin(ctx)
	require.NoError(t, err)

	sira := make(map[int64]int, len(list))
	for _, s := range list {
		sira[s.ID] = s.SortOrder
	}
	assert.Less(t, sira[ids[0]], sira[ids[1]])
	assert.Less(t, sira[ids[1]], sira[ids[2]])
}

func TestService_Reorder_VerilenSirayaGoreYenidenNumaralar(t *testing.T) {
	svc, _, ctx := newTestService(t)

	ids := ucSlayt(t, svc)

	require.NoError(t, svc.Reorder(ctx, []int64{ids[2], ids[0], ids[1]}))

	list, err := svc.ListAdmin(ctx)
	require.NoError(t, err)

	// ListAdmin sort_order'a göre sıralı döndüğü için sıra doğrudan okunur.
	require.Len(t, list, 3)
	assert.Equal(t, ids[2], list[0].ID)
	assert.Equal(t, ids[0], list[1].ID)
	assert.Equal(t, ids[1], list[2].ID)
}

func TestService_Reorder_EksikIDReddedilir(t *testing.T) {
	svc, _, ctx := newTestService(t)

	ids := ucSlayt(t, svc)

	err := svc.Reorder(ctx, []int64{ids[0], ids[1]})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Reorder_TekrarEdenIDReddedilir(t *testing.T) {
	svc, _, ctx := newTestService(t)

	ids := ucSlayt(t, svc)

	err := svc.Reorder(ctx, []int64{ids[0], ids[0], ids[1]})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}
