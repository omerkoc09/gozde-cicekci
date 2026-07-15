package idare

import "github.com/omerkoc/cicekci/internal/product"

// ProductView admin ürün gösterimi — is_active DAHİL.
type ProductView struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	Price       string  `json:"price"`
	IsActive    bool    `json:"is_active"`
	CategoryIDs []int64 `json:"category_ids"`
}

func toProductView(p product.Product) ProductView {
	return ProductView{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Price:       p.Price.StringFixed(2),
		IsActive:    p.IsActive,
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
