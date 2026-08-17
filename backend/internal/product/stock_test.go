package product

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// dec test için kısa decimal kurucusu. store_test.go'daki price() yardımcısı
// *testing.T alıyor; buradaki testler DB'siz olduğu için sade olanı kullanıyor.
func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

func intPtr(i int) *int { return &i }

func decPtr(s string) *decimal.Decimal {
	d := dec(s)
	return &d
}

func TestAvailable_TakipsizUrunSinirsiz(t *testing.T) {
	p := Product{TrackStock: false, StockQuantity: 0, StockReserved: 0}

	_, sinirli := p.Available()

	assert.False(t, sinirli, "takipsiz ürün sınırsız olmalı")
	assert.True(t, p.InStock(), "takipsiz ürün stokta 0 olsa bile satılabilir")
}

func TestAvailable_RezerveDusulur(t *testing.T) {
	p := Product{TrackStock: true, StockQuantity: 10, StockReserved: 3}

	adet, sinirli := p.Available()

	assert.True(t, sinirli)
	assert.Equal(t, 7, adet)
	assert.True(t, p.InStock())
}

func TestAvailable_TumuRezerveyseTukendi(t *testing.T) {
	p := Product{TrackStock: true, StockQuantity: 2, StockReserved: 2}

	adet, _ := p.Available()

	assert.Equal(t, 0, adet)
	assert.False(t, p.InStock(), "hepsi rezerveyse tükendi sayılmalı")
}

func TestAvailable_NegatifeDusmez(t *testing.T) {
	// Elle düzeltme sonrası rezerve stoktan büyük kalabilir; müşteriye
	// negatif adet göstermek yerine 0 döner.
	p := Product{TrackStock: true, StockQuantity: 1, StockReserved: 3}

	adet, _ := p.Available()

	assert.Equal(t, 0, adet)
}

func TestDiscountActive_KotaVarkenAktif(t *testing.T) {
	p := Product{Price: dec("1850.00"), DiscountPrice: decPtr("1450.00"),
		DiscountQuota: intPtr(10), DiscountSold: 3}

	assert.True(t, p.DiscountActive())
	assert.Equal(t, "1450", p.EffectivePrice().String())
	assert.Equal(t, "1850", p.OldPrice().String())
	assert.Equal(t, 7, *p.DiscountRemaining())
}

func TestDiscountActive_KotaDolunca_Soner(t *testing.T) {
	p := Product{Price: dec("1850.00"), DiscountPrice: decPtr("1450.00"),
		DiscountQuota: intPtr(10), DiscountSold: 10}

	assert.False(t, p.DiscountActive(), "kota dolunca indirim sönmeli")
	assert.Equal(t, "1850", p.EffectivePrice().String(), "normal fiyata dönmeli")
	assert.Nil(t, p.OldPrice(), "indirim yoksa eski fiyat gösterilmez")
	assert.Nil(t, p.DiscountRemaining())
}

func TestDiscountActive_IndirimYok(t *testing.T) {
	p := Product{Price: dec("1850.00")}

	assert.False(t, p.DiscountActive())
	assert.Equal(t, "1850", p.EffectivePrice().String())
	assert.Nil(t, p.OldPrice())
}
