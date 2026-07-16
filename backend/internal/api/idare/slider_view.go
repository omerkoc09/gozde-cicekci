package idare

import (
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/slider"
)

// SlideView admin slayt gösterimi — is_active DAHİL.
// Panelde önizleme 400'lük yeter, 1920'yi indirtmenin anlamı yok.
type SlideView struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Subtitle  string `json:"subtitle"`
	IsActive  bool   `json:"is_active"`
	SortOrder int    `json:"sort_order"`
	URL400    string `json:"url_400"`
	URL1200   string `json:"url_1200"`
}

func toSlideView(imgSvc *image.Service, s slider.Slide) SlideView {
	return SlideView{
		ID:        s.ID,
		Title:     s.Title,
		Subtitle:  s.Subtitle,
		IsActive:  s.IsActive,
		SortOrder: s.SortOrder,
		URL400:    imgSvc.URLFor(image.PrefixSlider, s.ImageKey, image.Size400),
		URL1200:   imgSvc.URLFor(image.PrefixSlider, s.ImageKey, image.Size1200),
	}
}

func toSlideViews(imgSvc *image.Service, list []slider.Slide) []SlideView {
	out := make([]SlideView, 0, len(list))
	for _, s := range list {
		out = append(out, toSlideView(imgSvc, s))
	}
	return out
}
