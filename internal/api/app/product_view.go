package app

import (
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
)

// ImageView public görsel gösterimi.
// id, sort_order ve image_key YOK — müşterinin işine yaramaz, iç detay sızmasın.
type ImageView struct {
	URL400  string `json:"url_400"`
	URL1200 string `json:"url_1200"`
}

// ProductView public ürün gösterimi.
// is_active alanı KASITLI olarak yok — public'e sızmaz (spec §4.6).
// Price string olarak gider: JSON float precision sorununu önler.
type ProductView struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	Description string      `json:"description"`
	Price       string      `json:"price"`
	CategoryIDs []int64     `json:"category_ids"`
	Images      []ImageView `json:"images"`
}

func toImageViews(imgSvc *image.Service, list []image.ProductImage) []ImageView {
	out := make([]ImageView, 0, len(list))
	for _, img := range list {
		out = append(out, ImageView{
			URL400:  imgSvc.URL(img.ImageKey, image.Size400),
			URL1200: imgSvc.URL(img.ImageKey, image.Size1200),
		})
	}
	return out
}

func toProductView(p product.Product, imgSvc *image.Service, imgs []image.ProductImage) ProductView {
	return ProductView{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Price:       p.Price.StringFixed(2),
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
