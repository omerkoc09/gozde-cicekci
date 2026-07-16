package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllowedOrigins_ProductionYalnizcaSiteURL(t *testing.T) {
	got := allowedOrigins("https://gozdetasarim.com", true)

	assert.Equal(t, "https://gozdetasarim.com", got)
	// Dev origin'i prod'a sızarsa herhangi bir yerel sayfa admin API'sine
	// cookie'li istek atabilirdi.
	assert.NotContains(t, got, "localhost")
}

func TestAllowedOrigins_DevelopmentAdminPaneliniIcerir(t *testing.T) {
	got := allowedOrigins("http://localhost:3000", false)

	assert.Contains(t, strings.Split(got, ","), "http://localhost:3000")
	assert.Contains(t, strings.Split(got, ","), "http://localhost:5173")
}
