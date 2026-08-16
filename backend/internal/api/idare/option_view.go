package idare

import "github.com/omerkoc/cicekci/internal/productoption"

// OptionValueView panel seçenek değeri — is_active DAHİL.
type OptionValueView struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SwatchHex string `json:"swatch_hex"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

// OptionGroupView panel seçenek grubu, değerleriyle birlikte.
type OptionGroupView struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	Slug      string            `json:"slug"`
	Kind      string            `json:"kind"`
	SortOrder int               `json:"sort_order"`
	IsActive  bool              `json:"is_active"`
	Values    []OptionValueView `json:"values"`
}

func toOptionValueView(v productoption.Value) OptionValueView {
	return OptionValueView{
		ID:        v.ID,
		Name:      v.Name,
		SwatchHex: v.SwatchHex,
		SortOrder: v.SortOrder,
		IsActive:  v.IsActive,
	}
}

func toOptionGroupView(g productoption.Group) OptionGroupView {
	values := make([]OptionValueView, 0, len(g.Values))
	for _, v := range g.Values {
		values = append(values, toOptionValueView(v))
	}
	return OptionGroupView{
		ID:        g.ID,
		Name:      g.Name,
		Slug:      g.Slug,
		Kind:      string(g.Kind),
		SortOrder: g.SortOrder,
		IsActive:  g.IsActive,
		Values:    values,
	}
}

func toOptionGroupViews(list []productoption.Group) []OptionGroupView {
	out := make([]OptionGroupView, 0, len(list))
	for _, g := range list {
		out = append(out, toOptionGroupView(g))
	}
	return out
}
