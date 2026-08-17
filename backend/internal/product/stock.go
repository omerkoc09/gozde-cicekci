package product

import "github.com/shopspring/decimal"

// Available satılabilir adedi ve bu sayının anlamlı olup olmadığını döner.
// sinirli=false → stok takibi kapalı, ürün sınırsız satılır (adet 0 döner
// ama anlamı yoktur).
//
// Rezerve edilmiş (ödeme bekleyen) adet düşülür: müşteri ödeme ekranındayken
// aynı ürünü başkası satın alamamalı.
func (p Product) Available() (adet int, sinirli bool) {
	if !p.TrackStock {
		return 0, false
	}
	kalan := p.StockQuantity - p.StockReserved
	// Elle düzeltme sonrası rezerve stoktan büyük kalabilir — negatif
	// göstermek yerine tükendi say.
	if kalan < 0 {
		return 0, true
	}
	return kalan, true
}

// InStock ürün satın alınabilir mi. Takipsiz ürün her zaman true.
func (p Product) InStock() bool {
	adet, sinirli := p.Available()
	return !sinirli || adet > 0
}

// DiscountActive indirim yürürlükte mi. Kota dolduğunda kendiliğinden
// false döner — ayrıca "indirimi kapat" işi çalıştırmaya gerek yok.
func (p Product) DiscountActive() bool {
	return p.DiscountPrice != nil && p.DiscountQuota != nil &&
		p.DiscountSold < *p.DiscountQuota
}

// EffectivePrice müşterinin ödeyeceği fiyat.
func (p Product) EffectivePrice() decimal.Decimal {
	if p.DiscountActive() {
		return *p.DiscountPrice
	}
	return p.Price
}

// OldPrice indirim aktifse üstü çizili gösterilecek normal fiyat.
func (p Product) OldPrice() *decimal.Decimal {
	if !p.DiscountActive() {
		return nil
	}
	fiyat := p.Price
	return &fiyat
}

// DiscountRemaining kalan indirimli adet; indirim yoksa nil.
func (p Product) DiscountRemaining() *int {
	if !p.DiscountActive() {
		return nil
	}
	kalan := *p.DiscountQuota - p.DiscountSold
	return &kalan
}
