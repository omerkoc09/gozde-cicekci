package customer

import "strings"

// normalizePhone Türkiye cep telefonu numarasını doğrular ve tek biçime
// getirir: 10 hane, başında 5 (örn. "5551112233").
//
// Kullanıcı numarayı çok farklı yazıyor — "0555 111 22 33", "+90 555 111
// 22 33", "(555) 111-2233". Hepsi aynı numara; veritabanında farklı
// biçimlerde durmaları arama ve karşılaştırmayı bozar. Bu yüzden önce
// biçimlendirme karakterleri (boşluk, tire, parantez, nokta) atılır, sonra
// ülke kodu / baştaki sıfır soyulur ve tek biçime indirgenir.
//
// ok=false ise numara geçersizdir: harf içeriyor, hane sayısı tutmuyor ya
// da 5 ile başlamıyor (sabit hat cep değildir — sipariş teslimatında
// müşteriye ulaşmak gerektiği için cep isteniyor).
func normalizePhone(raw string) (string, bool) {
	// Yalnızca rakamları al; yaygın biçimlendirme karakterlerini yok say.
	// Başka bir şey (harf vb.) varsa numara geçersizdir.
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-', r == '(', r == ')', r == '.', r == '/', r == '+':
			// biçimlendirme — yok say
		default:
			return "", false
		}
	}
	d := b.String()

	// Ülke kodu ve baştaki sıfırı soy: 90XXXXXXXXXX veya 0XXXXXXXXXX.
	switch {
	case len(d) == 12 && strings.HasPrefix(d, "90"):
		d = d[2:]
	case len(d) == 11 && strings.HasPrefix(d, "0"):
		d = d[1:]
	}

	// Geriye 5 ile başlayan 10 hane kalmalı.
	if len(d) != 10 || d[0] != '5' {
		return "", false
	}
	return d, true
}
