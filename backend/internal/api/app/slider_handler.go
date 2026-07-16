package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/slider"
)

type sliderHandler struct {
	svc    *slider.Service
	imgSvc *image.Service
}

// list GET /api/slides — ana sayfa slider'ı, sadece aktif slaytlar.
func (h *sliderHandler) list(c *fiber.Ctx) error {
	list, err := h.svc.ListPublic(c.Context())
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toSlideViews(h.imgSvc, list))
}
