package idare

import (
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/image"
)

// CategoryView admin kategori gösterimi — is_active ve is_featured DAHİL.
type CategoryView struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Axis       string `json:"axis"`
	IsActive   bool   `json:"is_active"`
	IsFeatured bool   `json:"is_featured"`
	SortOrder  int    `json:"sort_order"`

	// Kart görseli — yüklenmemişse boş string.
	URL400 string `json:"url_400"`
	URL900 string `json:"url_900"`
}

func toCategoryView(imgSvc *image.Service, c category.Category) CategoryView {
	v := CategoryView{
		ID:         c.ID,
		Name:       c.Name,
		Slug:       c.Slug,
		Axis:       string(c.Axis),
		IsActive:   c.IsActive,
		IsFeatured: c.IsFeatured,
		SortOrder:  c.SortOrder,
	}
	if c.ImageKey != "" {
		v.URL400 = imgSvc.URLFor(image.PrefixCategories, c.ImageKey, image.Size400)
		v.URL900 = imgSvc.URLFor(image.PrefixCategories, c.ImageKey, image.Size900)
	}
	return v
}

func toCategoryViews(imgSvc *image.Service, list []category.Category) []CategoryView {
	out := make([]CategoryView, 0, len(list))
	for _, c := range list {
		out = append(out, toCategoryView(imgSvc, c))
	}
	return out
}
