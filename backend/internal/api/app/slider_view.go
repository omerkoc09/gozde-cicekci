package app

import (
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/slider"
)

// SlideView public slayt gösterimi. is_active ve sort_order YOK —
// ziyaretçiyi ilgilendirmez; pasifler zaten hiç gelmiyor, sıra da
// dizinin kendisinde.
type SlideView struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	URL400   string `json:"url_400"`
	URL1200  string `json:"url_1200"`
	URL1920  string `json:"url_1920"`
}

func toSlideView(imgSvc *image.Service, s slider.Slide) SlideView {
	return SlideView{
		ID:       s.ID,
		Title:    s.Title,
		Subtitle: s.Subtitle,
		URL400:   imgSvc.URLFor(image.PrefixSlider, s.ImageKey, image.Size400),
		URL1200:  imgSvc.URLFor(image.PrefixSlider, s.ImageKey, image.Size1200),
		URL1920:  imgSvc.URLFor(image.PrefixSlider, s.ImageKey, image.Size1920),
	}
}

func toSlideViews(imgSvc *image.Service, list []slider.Slide) []SlideView {
	out := make([]SlideView, 0, len(list))
	for _, s := range list {
		out = append(out, toSlideView(imgSvc, s))
	}
	return out
}
