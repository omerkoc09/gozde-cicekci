package customer

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// CookieName müşteri JWT'sinin HttpOnly cookie adı. Admin cookie'sinden
// (cicekci_token) AYRI — iki oturum karışmaz.
const CookieName = "customer_token"

// Middleware müşteri cookie'sini doğrular. Geçerliyse Locals'a customerID koyar.
func Middleware(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies(CookieName)
		if token == "" {
			return api.WriteError(c, errorsx.ErrUnauthorized)
		}
		claims, err := ParseToken(token, secret)
		if err != nil {
			return api.WriteError(c, errorsx.ErrUnauthorized)
		}
		c.Locals("customerID", claims.CustomerID)
		return c.Next()
	}
}
