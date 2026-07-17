package order

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatOrderNo(t *testing.T) {
	d := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)

	assert.Equal(t, "2607-0001", FormatOrderNo(d, 1))
	assert.Equal(t, "2607-0042", FormatOrderNo(d, 42))
	assert.Equal(t, "2607-9999", FormatOrderNo(d, 9999))
}

func TestFormatOrderNo_AyGunSifirDolgulu(t *testing.T) {
	d := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)

	// 5 Ocak → "0501", tek haneli gün/ay sıfırla dolgulanmalı
	assert.Equal(t, "0501-0001", FormatOrderNo(d, 1))
}

func TestFormatOrderNo_DortHaneyiAsarsa(t *testing.T) {
	d := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)

	// Günde 10000+ sipariş gerçekçi değil ama format bozulmamalı
	assert.Equal(t, "2607-10000", FormatOrderNo(d, 10000))
}
