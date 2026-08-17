package idare

import (
	"time"

	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/internal/productoption"
)

// ProductView admin ürün gösterimi — is_active ve is_featured DAHİL.
type ProductView struct {
	ID           int64                    `json:"id"`
	Name         string                   `json:"name"`
	Slug         string                   `json:"slug"`
	Description  string                   `json:"description"`
	Price        string                   `json:"price"`
	IsActive     bool                     `json:"is_active"`
	IsFeatured   bool                     `json:"is_featured"`
	CategoryIDs  []int64                  `json:"category_ids"`
	Images       []ImageView              `json:"images"`
	OptionGroups []ProductOptionGroupView `json:"option_groups"`

	// Stok — panelde ham değerler gösterilir (public'te türetilmiş hali).
	TrackStock    bool `json:"track_stock"`
	StockQuantity int  `json:"stock_quantity"`
	StockReserved int  `json:"stock_reserved"`

	// İndirim. DiscountPrice null ise indirim yok.
	DiscountPrice *string `json:"discount_price"`
	DiscountQuota *int    `json:"discount_quota"`
	DiscountSold  int     `json:"discount_sold"`
}

// ProductOptionGroupView ürüne açık bir seçenek grubu. Panelde pasif
// gruplar da görünür (esnaf durumu anlasın).
type ProductOptionGroupView struct {
	ID       int64             `json:"id"`
	Name     string            `json:"name"`
	Kind     string            `json:"kind"`
	IsActive bool              `json:"is_active"`
	Values   []OptionValueView `json:"values"`
}

func toProductOptionGroupViews(list []productoption.ProductGroup) []ProductOptionGroupView {
	out := make([]ProductOptionGroupView, 0, len(list))
	for _, g := range list {
		values := make([]OptionValueView, 0, len(g.Values))
		for _, v := range g.Values {
			values = append(values, toOptionValueView(v))
		}
		out = append(out, ProductOptionGroupView{
			ID:       g.ID,
			Name:     g.Name,
			Kind:     string(g.Kind),
			IsActive: g.IsActive,
			Values:   values,
		})
	}
	return out
}

func toProductView(p product.Product, imgSvc *image.Service, imgs []image.ProductImage) ProductView {
	// Price panelde NORMAL fiyat — esnaf indirimli fiyatı ayrı alanda
	// düzenliyor, ana fiyat alanı indirimden etkilenmemeli.
	v := ProductView{
		ID:            p.ID,
		Name:          p.Name,
		Slug:          p.Slug,
		Description:   p.Description,
		Price:         p.Price.StringFixed(2),
		IsActive:      p.IsActive,
		IsFeatured:    p.IsFeatured,
		CategoryIDs:   p.CategoryIDs,
		Images:        toImageViews(imgSvc, imgs),
		TrackStock:    p.TrackStock,
		StockQuantity: p.StockQuantity,
		StockReserved: p.StockReserved,
		DiscountQuota: p.DiscountQuota,
		DiscountSold:  p.DiscountSold,
	}
	if p.DiscountPrice != nil {
		s := p.DiscountPrice.StringFixed(2)
		v.DiscountPrice = &s
	}
	return v
}

// toProductViews N+1 sorgu yapmaz — görseller tek sorguda gelir (grouped).
func toProductViews(list []product.Product, imgSvc *image.Service,
	grouped map[int64][]image.ProductImage) []ProductView {
	out := make([]ProductView, 0, len(list))
	for _, p := range list {
		out = append(out, toProductView(p, imgSvc, grouped[p.ID]))
	}
	return out
}

// MovementView panelde gösterilen stok hareketi. Sebep kodu Türkçe etikete
// frontend'de çevriliyor — kod burada ham kalıyor ki filtrelenebilsin.
type MovementView struct {
	ID            int64  `json:"id"`
	Delta         int    `json:"delta"`
	Reason        string `json:"reason"`
	OrderID       *int64 `json:"order_id"`
	WasDiscounted bool   `json:"was_discounted"`
	Note          string `json:"note"`
	CreatedAt     string `json:"created_at"`
}

func toMovementViews(list []product.Movement) []MovementView {
	out := make([]MovementView, 0, len(list))
	for _, m := range list {
		out = append(out, MovementView{
			ID:            m.ID,
			Delta:         m.Delta,
			Reason:        string(m.Reason),
			OrderID:       m.OrderID,
			WasDiscounted: m.WasDiscounted,
			Note:          m.Note,
			CreatedAt:     m.CreatedAt.Format(time.RFC3339),
		})
	}
	return out
}
