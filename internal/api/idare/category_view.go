package idare

import "github.com/omerkoc/cicekci/internal/category"

// CategoryView admin kategori gösterimi — is_active ve is_featured DAHİL.
type CategoryView struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Axis       string `json:"axis"`
	IsActive   bool   `json:"is_active"`
	IsFeatured bool   `json:"is_featured"`
	SortOrder  int    `json:"sort_order"`
}

func toCategoryView(c category.Category) CategoryView {
	return CategoryView{
		ID:         c.ID,
		Name:       c.Name,
		Slug:       c.Slug,
		Axis:       string(c.Axis),
		IsActive:   c.IsActive,
		IsFeatured: c.IsFeatured,
		SortOrder:  c.SortOrder,
	}
}

func toCategoryViews(list []category.Category) []CategoryView {
	out := make([]CategoryView, 0, len(list))
	for _, c := range list {
		out = append(out, toCategoryView(c))
	}
	return out
}
