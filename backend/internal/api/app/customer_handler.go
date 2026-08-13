package app

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/customer"
	"github.com/omerkoc/cicekci/internal/order"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// errGecersizIstek gövde parse edilemediğinde dönülür. app paketinde
// badRequest helper'ı YOK (o idare'de) — desen api.WriteError + ErrInvalidInput.
var errGecersizIstek = fmt.Errorf("%w: geçersiz istek", errorsx.ErrInvalidInput)

type customerHandler struct {
	svc          *customer.Service
	orderSvc     *order.Service
	secureCookie bool
}

func (h *customerHandler) setCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     customer.CookieName,
		Value:    token,
		HTTPOnly: true,
		Secure:   h.secureCookie,
		SameSite: "Strict",
		Path:     "/",
		Expires:  time.Now().Add(customer.TokenTTL),
	})
}

func (h *customerHandler) register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return api.WriteError(c, errGecersizIstek)
	}
	token, cst, err := h.svc.Register(c.Context(), req.Email, req.Password, req.Name, req.Phone)
	if err != nil {
		return api.WriteError(c, err)
	}
	h.setCookie(c, token)
	return c.Status(fiber.StatusCreated).JSON(toCustomerView(cst))
}

func (h *customerHandler) login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return api.WriteError(c, errGecersizIstek)
	}
	token, err := h.svc.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return api.WriteError(c, err)
	}
	h.setCookie(c, token)
	return c.JSON(fiber.Map{"ok": true})
}

func (h *customerHandler) logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name: customer.CookieName, Value: "", HTTPOnly: true,
		Secure: h.secureCookie, SameSite: "Strict", Path: "/",
		Expires: time.Now().Add(-time.Hour),
	})
	return c.JSON(fiber.Map{"ok": true})
}

func (h *customerHandler) me(c *fiber.Ctx) error {
	id := c.Locals("customerID").(int64)
	cst, err := h.svc.Get(c.Context(), id)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCustomerView(cst))
}

func (h *customerHandler) updateMe(c *fiber.Ctx) error {
	id := c.Locals("customerID").(int64)
	var req updateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return api.WriteError(c, errGecersizIstek)
	}
	// Şifre değiştirme istendiyse önce onu uygula.
	if req.NewPassword != "" {
		if err := h.svc.ChangePassword(c.Context(), id, req.CurrentPassword, req.NewPassword); err != nil {
			return api.WriteError(c, err)
		}
	}
	cst, err := h.svc.UpdateProfile(c.Context(), id, req.Name, req.Phone)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCustomerView(cst))
}

func (h *customerHandler) orders(c *fiber.Ctx) error {
	id := c.Locals("customerID").(int64)
	list, err := h.orderSvc.ListByCustomer(c.Context(), id)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toCreateOrderCustomerViews(list))
}
