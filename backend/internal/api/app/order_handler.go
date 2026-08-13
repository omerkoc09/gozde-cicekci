package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/customer"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/internal/payment"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

var errInvalidDate = fmt.Errorf("%w: geçersiz teslimat tarihi", errorsx.ErrInvalidInput)

type orderHandler struct {
	svc       *order.Service
	cfg       order.DeliveryConfig
	jwtSecret string
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
		BuyerName:        req.Buyer.Name,
		BuyerPhone:       req.Buyer.Phone,
		BuyerEmail:       req.Buyer.Email,
		RecipientName:    req.Recipient.Name,
		RecipientPhone:   req.Recipient.Phone,
		DeliveryAddress:  req.Delivery.Address,
		DeliveryDistrict: req.Delivery.District,
		DeliveryDate:     date,
		DeliverySlot:     req.Delivery.Slot,
		CardMessage:      req.CardMessage,
	}
	for _, it := range req.Items {
		in.Items = append(in.Items, order.CreateItem{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
		})
	}

	// Giriş yapmış müşteri varsa siparişi ona bağla (opsiyonel — yoksa misafir).
	var customerID *int64
	if tok := c.Cookies(customer.CookieName); tok != "" {
		if claims, err := customer.ParseToken(tok, h.jwtSecret); err == nil {
			customerID = &claims.CustomerID
		}
	}

	o, token, err := h.svc.Create(c.Context(), in, c.IP(), customerID)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toCreateOrderResponse(o, token))
}

// paymentCallback PayTR sunucu-sunucu bildirimi. Yanıt DÜZ METİN "OK" olmalı,
// yoksa PayTR tekrar tekrar gönderir. Ödeme kararı YALNIZCA burada verilir.
func (h *orderHandler) paymentCallback(c *fiber.Ctx) error {
	in := payment.CallbackInput{
		MerchantOID: c.FormValue("merchant_oid"),
		Status:      c.FormValue("status"),
		TotalAmount: c.FormValue("total_amount"),
		Hash:        c.FormValue("hash"),
	}
	// raw_payload JSONB kolonu — PayTR'nin gönderdiği ham gövde form-encoded
	// (merchant_oid=...&status=...), GEÇERLİ JSON DEĞİL. Ham gövdeyi doğrudan
	// geçmek JSONB insert'ini "invalid input syntax for type json" ile
	// reddeder; bu hata AddPaymentEvent'te yutulduğu için callback_ok hiç
	// yazılmaz ve idempotency kontrolü (HasPaymentEvent) bozulur. Bunun yerine
	// parse edilmiş alanları JSON'a çevirip yapılandırılmış + geçerli bir
	// denetim izi payload'ı üretiyoruz.
	raw, err := json.Marshal(in)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("FAIL")
	}

	accepted, err := h.svc.ApplyCallback(c.Context(), in, raw)
	if err != nil || !accepted {
		// Hata/geçersiz → PayTR'ye OK DÖNME (tekrar denesin / sahte reddedilsin).
		// PayTR "OK dışı" yanıtı başarısız sayar.
		return c.Status(fiber.StatusBadRequest).SendString("FAIL")
	}
	return c.SendString("OK")
}

func (h *orderHandler) deliveryConfig(c *fiber.Ctx) error {
	return c.JSON(deliveryConfigResponse{
		Fee:           h.cfg.Fee,
		Slots:         h.cfg.Slots,
		SameDayCutoff: h.cfg.SameDayCutoff,
		MaxDays:       h.cfg.MaxDays,
		Districts:     h.cfg.Districts,
		DistrictFees:  h.cfg.DistrictFees,
	})
}
