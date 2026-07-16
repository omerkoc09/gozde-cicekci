package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// CookieName — JWT HttpOnly cookie'de tutulur, localStorage'da değil (spec §4.5).
const CookieName = "cicekci_token"

// Middleware JWT cookie'sini doğrular. Geçerliyse Locals'a user bilgisi koyar.
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

		c.Locals("userID", claims.UserID)
		c.Locals("username", claims.Username)
		return c.Next()
	}
}
