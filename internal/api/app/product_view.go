package app

import "github.com/omerkoc/cicekci/internal/product"

// ProductView public ürün gösterimi.
// is_active alanı KASITLI olarak yok — public'e sızmaz (spec §4.6).
// Price string olarak gider: JSON float precision sorununu önler.
type ProductView struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	Price       string  `json:"price"`
	CategoryIDs []int64 `json:"category_ids"`
}

func toProductView(p product.Product) ProductView {
	return ProductView{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Price:       p.Price.StringFixed(2),
		CategoryIDs: p.CategoryIDs,
	}
}

func toProductViews(list []product.Product) []ProductView {
	out := make([]ProductView, 0, len(list))
	for _, p := range list {
		out = append(out, toProductView(p))
	}
	return out
}
