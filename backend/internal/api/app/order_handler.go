package app

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

var errInvalidDate = fmt.Errorf("%w: geçersiz teslimat tarihi", errorsx.ErrInvalidInput)

type orderHandler struct {
	svc *order.Service
	cfg order.DeliveryConfig
}

func (h *orderHandler) create(c *fiber.Ctx) error {
	var req createOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return api.WriteError(c, err)
	}

	date, err := time.Parse("2006-01-02", req.Delivery.Date)
	if err != nil {
		return api.WriteError(c, errInvalidDate)
	}

	in := order.CreateInput{
		BuyerName:       req.Buyer.Name,
		BuyerPhone:      req.Buyer.Phone,
		BuyerEmail:      req.Buyer.Email,
		RecipientName:   req.Recipient.Name,
		RecipientPhone:  req.Recipient.Phone,
		DeliveryAddress: req.Delivery.Address,
		DeliveryDate:    date,
		DeliverySlot:    req.Delivery.Slot,
		CardMessage:     req.CardMessage,
	}
	for _, it := range req.Items {
		in.Items = append(in.Items, order.CreateItem{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
		})
	}

	o, err := h.svc.Create(c.Context(), in)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toCreateOrderResponse(o))
}

func (h *orderHandler) deliveryConfig(c *fiber.Ctx) error {
	return c.JSON(deliveryConfigResponse{
		Fee:           h.cfg.Fee,
		Slots:         h.cfg.Slots,
		SameDayCutoff: h.cfg.SameDayCutoff,
		MaxDays:       h.cfg.MaxDays,
	})
}
