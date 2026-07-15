package idare

import (
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
)

// ProductView admin ürün gösterimi — is_active DAHİL.
type ProductView struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	Description string      `json:"description"`
	Price       string      `json:"price"`
	IsActive    bool        `json:"is_active"`
	CategoryIDs []int64     `json:"category_ids"`
	Images      []ImageView `json:"images"`
}

func toProductView(p product.Product, imgSvc *image.Service, imgs []image.ProductImage) ProductView {
	return ProductView{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Price:       p.Price.StringFixed(2),
		IsActive:    p.IsActive,
		CategoryIDs: p.CategoryIDs,
		Images:      toImageViews(imgSvc, imgs),
	}
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
