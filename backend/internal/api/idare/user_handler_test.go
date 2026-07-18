package idare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdmin_UsersRequireAuth(t *testing.T) {
	app, _ := newTestAdminAPI(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/admin/users", nil))

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdmin_ListUsers(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(authedRequest(http.MethodGet, "/api/admin/users", "", token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var users []adminUserView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&users))
	require.Len(t, users, 1)
	assert.Equal(t, "cicekci", users[0].Username)
}

func TestAdmin_CreateUser(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/users",
		`{"username":"yenieleman","password":"yeni-sifre-123"}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	listResp, err := app.Test(authedRequest(http.MethodGet, "/api/admin/users", "", token))
	require.NoError(t, err)
	var users []adminUserView
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&users))
	assert.Len(t, users, 2)
}

func TestAdmin_CreateUser_DuplicateUsername(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/users",
		`{"username":"cicekci","password":"baska-sifre-123"}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestAdmin_DeleteUser_CannotDeleteSelf(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(authedRequest(http.MethodDelete, "/api/admin/users/1", "", token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAdmin_DeleteUser_Success(t *testing.T) {
	app, token := newTestAdminAPI(t)

	createResp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/users",
		`{"username":"silinecek","password":"yeni-sifre-123"}`, token))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	resp, err := app.Test(authedRequest(http.MethodDelete, "/api/admin/users/2", "", token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestAdmin_ResetPassword_ShortPassword(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(authedRequest(http.MethodPatch, "/api/admin/users/1/password",
		`{"password":"kisa"}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAdmin_ResetPassword_Success(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(authedRequest(http.MethodPatch, "/api/admin/users/1/password",
		`{"password":"yeni-sifre-456"}`, token))

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	loginResp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/login",
		`{"username":"cicekci","password":"yeni-sifre-456"}`, ""))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)
}
