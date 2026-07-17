package idare

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/order"
)

type orderHandler struct {
	svc *order.Service
}

func (h *orderHandler) list(c *fiber.Ctx) error {
	status := c.Query("status")
	limit := c.QueryInt("limit", 50)
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}

	list, err := h.svc.List(c.Context(), status, limit, (page-1)*limit)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(toOrderViews(list))
}

func (h *orderHandler) get(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	o, err := h.svc.Get(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(toOrderView(o))
}

func (h *orderHandler) update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req updateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	o, err := h.svc.Update(c.Context(), int64(id), req.Status, req.Note)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(toOrderView(o))
}
