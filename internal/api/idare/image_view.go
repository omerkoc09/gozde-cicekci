package idare

import "github.com/omerkoc/cicekci/internal/image"

// ImageView admin görsel gösterimi.
// image_key JSON'a çıkmaz — URL'ler ImageStore'dan üretilir (spec §4.1).
type ImageView struct {
	ID        int64  `json:"id"`
	URL400    string `json:"url_400"`
	URL1200   string `json:"url_1200"`
	SortOrder int    `json:"sort_order"`
}

func toImageView(svc *image.Service, img image.ProductImage) ImageView {
	return ImageView{
		ID:        img.ID,
		URL400:    svc.URL(img.ImageKey, image.Size400),
		URL1200:   svc.URL(img.ImageKey, image.Size1200),
		SortOrder: img.SortOrder,
	}
}

func toImageViews(svc *image.Service, list []image.ProductImage) []ImageView {
	out := make([]ImageView, 0, len(list))
	for _, img := range list {
		out = append(out, toImageView(svc, img))
	}
	return out
}
