package idare

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/auth"
)

type authHandler struct {
	svc          *auth.Service
	secureCookie bool
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// login POST /api/admin/login
// Token HttpOnly cookie'ye yazılır, response body'de dönmez (spec §4.5).
func (h *authHandler) login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
			Error: api.ErrorBody{Code: "invalid_input", Message: "Geçersiz istek"},
		})
	}

	token, err := h.svc.Login(c.Context(), req.Username, req.Password)
	if err != nil {
		return api.WriteError(c, err)
	}

	c.Cookie(&fiber.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		HTTPOnly: true,
		Secure:   h.secureCookie,
		SameSite: "Strict",
		Path:     "/",
		Expires:  time.Now().Add(auth.TokenTTL),
	})

	return c.JSON(fiber.Map{"ok": true})
}

// logout POST /api/admin/logout
func (h *authHandler) logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		HTTPOnly: true,
		Secure:   h.secureCookie,
		SameSite: "Strict",
		Path:     "/",
		Expires:  time.Now().Add(-time.Hour),
	})
	return c.JSON(fiber.Map{"ok": true})
}

// me GET /api/admin/me — frontend'in oturum kontrolü için
func (h *authHandler) me(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"username": c.Locals("username")})
}
