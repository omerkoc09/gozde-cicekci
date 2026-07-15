package app

import "github.com/omerkoc/cicekci/internal/category"

// CategoryView public kategori gösterimi.
// is_active ve is_featured alanları KASITLI olarak yok — public'e sızmaz (spec §4.6).
type CategoryView struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Axis string `json:"axis"`
}

func toCategoryView(c category.Category) CategoryView {
	return CategoryView{
		ID:   c.ID,
		Name: c.Name,
		Slug: c.Slug,
		Axis: string(c.Axis),
	}
}

func toCategoryViews(list []category.Category) []CategoryView {
	out := make([]CategoryView, 0, len(list))
	for _, c := range list {
		out = append(out, toCategoryView(c))
	}
	return out
}
