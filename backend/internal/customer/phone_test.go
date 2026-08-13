package customer

import "testing"

func TestNormalizePhone_GecerliBicimler(t *testing.T) {
	// Hepsi AYNI numara — kullanıcının yazdığı biçim ne olursa olsun
	// veritabanına tek biçimde yazılmalı.
	durumlar := []struct {
		girdi    string
		beklenen string
	}{
		{"5551112233", "5551112233"},
		{"05551112233", "5551112233"},
		{"905551112233", "5551112233"},
		{"+905551112233", "5551112233"},
		{"0555 111 22 33", "5551112233"},
		{"+90 555 111 22 33", "5551112233"},
		{"(555) 111-2233", "5551112233"},
		{"555.111.2233", "5551112233"},
		{"  5551112233  ", "5551112233"},
	}

	for _, d := range durumlar {
		got, ok := normalizePhone(d.girdi)
		if !ok {
			t.Errorf("normalizePhone(%q) reddedildi, geçerli olmalıydı", d.girdi)
			continue
		}
		if got != d.beklenen {
			t.Errorf("normalizePhone(%q) = %q, beklenen %q", d.girdi, got, d.beklenen)
		}
	}
}

func TestNormalizePhone_GecersizReddedilir(t *testing.T) {
	gecersiz := []struct {
		girdi  string
		neden  string
	}{
		{"asdasd", "harf"},
		{"555111223a", "sonda harf"},
		{"abc5551112233", "başta harf"},
		{"", "boş"},
		{"   ", "yalnız boşluk"},
		{"555111223", "9 hane — eksik"},
		{"55511122334", "11 hane — fazla"},
		{"2121112233", "sabit hat (5 ile başlamıyor)"},
		{"4441112233", "kurumsal hat"},
		{"0000000000", "5 ile başlamıyor"},
		{"5551112233@", "özel karakter"},
		{"555-111-22-3x", "sonda harf"},
	}

	for _, d := range gecersiz {
		if got, ok := normalizePhone(d.girdi); ok {
			t.Errorf("normalizePhone(%q) kabul edildi (%q), reddedilmeliydi — %s", d.girdi, got, d.neden)
		}
	}
}
