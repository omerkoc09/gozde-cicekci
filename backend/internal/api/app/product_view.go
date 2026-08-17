package app

import (
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/internal/productoption"
)

// ImageView public görsel gösterimi.
// id, sort_order ve image_key YOK — müşterinin işine yaramaz, iç detay sızmasın.
type ImageView struct {
	URL400  string `json:"url_400"`
	URL800  string `json:"url_800"`
	URL1200 string `json:"url_1200"`
}

// ProductView public ürün gösterimi.
// is_active alanı KASITLI olarak yok — public'e sızmaz (spec §4.6).
// Price string olarak gider: JSON float precision sorununu önler.
type ProductView struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	// Price GEÇERLİ fiyat — indirim aktifse indirimli fiyat. Sepet ve toplam
	// hesapları bu alanı kullanır, değişiklik gerektirmez.
	Price string `json:"price"`
	// OldPrice indirim aktifse üstü çizili gösterilecek normal fiyat.
	OldPrice *string `json:"old_price"`
	// InStock takipsiz üründe her zaman true.
	InStock bool `json:"in_stock"`
	// StockQuantity takipsiz üründe null — adet bilgisi anlamsız.
	StockQuantity *int `json:"stock_quantity"`
	// DiscountRemaining kalan indirimli adet; indirim yoksa null.
	DiscountRemaining *int                    `json:"discount_remaining"`
	CategoryIDs       []int64                 `json:"category_ids"`
	Images            []ImageView             `json:"images"`
	OptionGroups      []PublicOptionGroupView `json:"option_groups"`
}

// PublicOptionValueView müşteriye görünen seçenek değeri.
// is_active YOK — pasif değerler bu uçtan zaten hiç gelmez.
type PublicOptionValueView struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SwatchHex string `json:"swatch_hex"`
}

// PublicOptionGroupView müşteriye görünen seçenek grubu.
type PublicOptionGroupView struct {
	ID     int64                   `json:"id"`
	Name   string                  `json:"name"`
	Kind   string                  `json:"kind"`
	Values []PublicOptionValueView `json:"values"`
}

func toPublicOptionGroupViews(list []productoption.ProductGroup) []PublicOptionGroupView {
	out := make([]PublicOptionGroupView, 0, len(list))
	for _, g := range list {
		// Değeri kalmamış grup müşteriye gösterilmez — seçenek sunmayan
		// bir başlık kafa karıştırır.
		if len(g.Values) == 0 {
			continue
		}
		values := make([]PublicOptionValueView, 0, len(g.Values))
		for _, v := range g.Values {
			values = append(values, PublicOptionValueView{
				ID: v.ID, Name: v.Name, SwatchHex: v.SwatchHex,
			})
		}
		out = append(out, PublicOptionGroupView{
			ID:     g.ID,
			Name:   g.Name,
			Kind:   string(g.Kind),
			Values: values,
		})
	}
	return out
}

func toImageViews(imgSvc *image.Service, list []image.ProductImage) []ImageView {
	out := make([]ImageView, 0, len(list))
	for _, img := range list {
		out = append(out, ImageView{
			URL400:  imgSvc.URL(img.ImageKey, image.Size400),
			URL800:  imgSvc.URL(img.ImageKey, image.Size800),
			URL1200: imgSvc.URL(img.ImageKey, image.Size1200),
		})
	}
	return out
}

func toProductView(p product.Product, imgSvc *image.Service, imgs []image.ProductImage) ProductView {
	v := ProductView{
		ID:                p.ID,
		Name:              p.Name,
		Slug:              p.Slug,
		Description:       p.Description,
		Price:             p.EffectivePrice().StringFixed(2),
		InStock:           p.InStock(),
		DiscountRemaining: p.DiscountRemaining(),
		CategoryIDs:       p.CategoryIDs,
		Images:            toImageViews(imgSvc, imgs),
	}
	if old := p.OldPrice(); old != nil {
		s := old.StringFixed(2)
		v.OldPrice = &s
	}
	// Takipsiz üründe adet anlamsız — null gider.
	if adet, sinirli := p.Available(); sinirli {
		v.StockQuantity = &adet
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
