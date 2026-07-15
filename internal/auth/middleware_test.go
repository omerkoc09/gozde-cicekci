package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Get("/korumali", Middleware(testSecret), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"username": c.Locals("username")})
	})
	return app
}

func TestMiddleware_NoCookie(t *testing.T) {
	app := newTestApp(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/korumali", nil))

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMiddleware_ValidToken(t *testing.T) {
	app := newTestApp(t)
	token, err := GenerateToken(1, "cicekci", testSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/korumali", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestMiddleware_InvalidToken(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/korumali", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "bu-token-degil"})
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMiddleware_TokenSignedWithOtherSecret(t *testing.T) {
	app := newTestApp(t)
	token, err := GenerateToken(1, "saldirgan", "baska-secret-yeterince-uzun-32ch")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/korumali", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
