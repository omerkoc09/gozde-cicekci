package idare

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Admin uçları JWT ister — cookie'siz istek 401 dönmeli.
func TestOrders_AuthGerekli(t *testing.T) {
	app, _ := newTestAdminAPI(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/admin/orders", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
