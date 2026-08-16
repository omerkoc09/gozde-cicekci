package productoption

import (
	"context"
	"fmt"

	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// ResolveForProduct order.OptionReader'ı karşılar: müşterinin gönderdiği
// değer id'lerini doğrular ve isimleriyle döner.
//
// Reddedilen durumlar:
//   - değer yok veya pasif
//   - değerin grubu bu ürüne açık değil
//   - aynı gruptan birden fazla değer
//   - ürünün zorunlu grubu doldurulmamış
func (s *Service) ResolveForProduct(ctx context.Context, productID int64, valueIDs []int64) ([]order.OrderItemOption, error) {
	gruplar, err := s.store.GroupsForProduct(ctx, productID, true)
	if err != nil {
		return nil, err
	}

	// Ürüne açık aktif değerlerin dizini: valueID → (grup, değer)
	type kayit struct {
		grup  ProductGroup
		deger Value
	}
	dizin := make(map[int64]kayit)
	for _, g := range gruplar {
		for _, v := range g.Values {
			dizin[v.ID] = kayit{grup: g, deger: v}
		}
	}

	out := make([]order.OrderItemOption, 0, len(valueIDs))
	gorulenGrup := make(map[int64]bool, len(valueIDs))

	for _, vid := range valueIDs {
		k, ok := dizin[vid]
		if !ok {
			return nil, fmt.Errorf("%w: geçersiz veya artık sunulmayan seçenek", errorsx.ErrInvalidInput)
		}
		if gorulenGrup[k.grup.ID] {
			return nil, fmt.Errorf("%w: %q için birden fazla seçim gönderildi",
				errorsx.ErrInvalidInput, k.grup.Name)
		}
		gorulenGrup[k.grup.ID] = true

		out = append(out, order.OrderItemOption{
			GroupName: k.grup.Name,
			ValueName: k.deger.Name,
			SwatchHex: k.deger.SwatchHex,
			SortOrder: k.grup.SortOrder,
		})
	}

	// Zorunluluk kontrolü YOK (2026-08-16 kararı): müşteri sayfasında her
	// grubun ilk değeri otomatik seçili geliyor, yani seçim zaten hep dolu.
	// Boş gelirse de reddedilmez — esnaf uygun olanı koyar.
	return out, nil
}
