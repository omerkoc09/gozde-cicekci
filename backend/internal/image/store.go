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
	Size400  Size = "400"  // liste kartları (mobil / düşük yoğunluk)
	Size800  Size = "800"  // ürün kartı retina (2x) — 4:5 grid'de keskin
	Size900  Size = "900"  // kategori kartı (3:4 dikey, retina için 2x)
	Size1200 Size = "1200" // detay sayfası + og:image
	Size1920 Size = "1920" // ana sayfa slider — tam genişlik arka plan
)

// AllSizes ürün görselleri için üretilen boyutlar. 800: kart retina (2x)
// keskinliği için — sadece 400 varken kartlar yüksek yoğunluklu ekranda
// bulanık kalıyordu (kaynak yatay foto 4:5 karta kırpılıp büyütülünce).
var AllSizes = []Size{Size400, Size800, Size1200}

// SliderSizes slider görselleri için üretilen boyutlar. Slider tam genişlik
// gösteriliyor; 1200 geniş ekranda bulanık kalır, o yüzden 1920 de üretilir.
// 400 mobil için: 1920'lik dosyayı telefona indirtmek israf.
var SliderSizes = []Size{Size400, Size1200, Size1920}

// CategorySizes kategori kartı boyutları. Kart 4'lü grid'de en fazla ~450px
// görünüyor; 900 retina (2x) için yeterli, daha büyüğü boşuna bayt.
var CategorySizes = []Size{Size400, Size900}

// Width piksel karşılığı.
func (s Size) Width() int {
	switch s {
	case Size400:
		return 400
	case Size800:
		return 800
	case Size900:
		return 900
	case Size1200:
		return 1200
	case Size1920:
		return 1920
	default:
		return 0
	}
}

func (s Size) Valid() bool {
	return s.Width() > 0
}

// Prefix saklamadaki üst klasör — görsel türlerini ayırır.
// Key'ler rastgele olduğu için çakışma riski yok; ayrım operasyonel:
// bucket'a bakınca neyin ne olduğu görünsün, biri silinirken diğeri etkilenmesin.
type Prefix string

const (
	PrefixProducts   Prefix = "products"
	PrefixSlider     Prefix = "slider"
	PrefixCategories Prefix = "categories"
)

// Store görsel saklamayı soyutlar. Mimari kısıt (spec §4.4): bugün R2,
// yarın disk — uygulama kodu değişmez. Bu interface Fiber'i, HTTP'yi ve
// veritabanını bilmez.
type Store interface {
	// Put bir boyutu yazar. Aynı key+size varsa üzerine yazar.
	Put(ctx context.Context, prefix Prefix, key string, size Size, data []byte) error

	// Delete bir key'in verilen boyutlardaki dosyalarını siler.
	Delete(ctx context.Context, prefix Prefix, key string, sizes []Size) error

	// URL public erişim adresini üretir. Domain veritabanında değil,
	// burada yaşar (spec §4.1).
	URL(prefix Prefix, key string, size Size) string
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

// objectPath key ve boyuttan saklama yolunu üretir: "products/a3f8c2d1/400.jpg"
func objectPath(prefix Prefix, key string, size Size) string {
	return fmt.Sprintf("%s/%s/%s.jpg", prefix, key, size)
}
