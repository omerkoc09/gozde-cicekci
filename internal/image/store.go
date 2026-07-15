package image

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Size servis edilen görsel boyutları. Orijinal saklanmaz (spec §4.4).
type Size string

const (
	Size400  Size = "400"  // liste kartları
	Size1200 Size = "1200" // detay sayfası + og:image
)

// AllSizes yükleme sırasında üretilen tüm boyutlar.
var AllSizes = []Size{Size400, Size1200}

// Width piksel karşılığı.
func (s Size) Width() int {
	switch s {
	case Size400:
		return 400
	case Size1200:
		return 1200
	default:
		return 0
	}
}

func (s Size) Valid() bool {
	return s.Width() > 0
}

// Store görsel saklamayı soyutlar. Mimari kısıt (spec §4.4): bugün R2,
// yarın disk — uygulama kodu değişmez. Bu interface Fiber'i, HTTP'yi ve
// veritabanını bilmez.
type Store interface {
	// Put bir boyutu yazar. Aynı key+size varsa üzerine yazar.
	Put(ctx context.Context, key string, size Size, data []byte) error

	// Delete bir key'in TÜM boyutlarını siler.
	Delete(ctx context.Context, key string) error

	// URL public erişim adresini üretir. Domain veritabanında değil,
	// burada yaşar (spec §4.1).
	URL(key string, size Size) string
}

// NewKey çakışmayan rastgele bir key üretir: "a3f8c2d1e9b40726".
// Rastgele olması önemli — key'ler immutable cache'leniyor, aynı key'in
// içeriği asla değişmemeli.
func NewKey() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("rastgele key üretilemedi: %v", err))
	}
	return hex.EncodeToString(b)
}

// objectPath key ve boyuttan saklama yolunu üretir: "products/a3f8c2d1/400.webp"
func objectPath(key string, size Size) string {
	return fmt.Sprintf("products/%s/%s.webp", key, size)
}
