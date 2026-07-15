# Plan 2 — Görsel Hattı Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Esnafın yüklediği JPEG/PNG/WebP fotoğrafı iki boyutta WebP'ye çevirip R2'ye yazan, ürüne bağlayan, sıralayan ve silen tam bir görsel hattı.

**Architecture:** `ImageStore` interface'i saklamayı soyutlar — `r2.go` (production), `local.go` (geliştirme/test). İşleme (`process.go`) saklamadan bağımsız: `[]byte` alır, `[]byte` verir. `image.Service` ikisini birleştirir ve `product_images` tablosunu yönetir. Handler'lar `internal/api/idare/` altında.

**Tech Stack:** `disintegration/imaging` (resize), `gen2brain/webp` (saf Go WebP encode, cgo yok), `aws-sdk-go-v2` (R2 S3-uyumlu API), `golang.org/x/image/webp` (WebP decode).

**Spec:** `docs/superpowers/specs/2026-07-15-cicekci-mvp-design.md` — bu plan §4.4'ü ve §4.6'nın görsel uçlarını kapsar.

**Önkoşul:** Plan 1 tamamlanmış olmalı — `products` tablosu, `product.Service`, `auth.Middleware`, `api.WriteError` mevcut.

## Global Constraints

- **Girdi:** JPEG, PNG, WebP kabul edilir. Format **içerikten** tespit edilir (`http.DetectContentType`), uzantıdan değil — uzantı yalan söyleyebilir.
- **Çıktı:** Sadece WebP. JPEG fallback yok (spec §4.4).
- **İki boyut:** `Size400` (liste kartları), `Size1200` (detay + `og:image`). Orijinal saklanmaz.
- **DB'de `image_key`, tam URL değil.** URL'i `ImageStore.URL()` üretir. Domain veritabanına sızmaz (spec §4.1).
- **Atomiklik:** Her iki boyut da R2'ye yazılmadan DB'ye kayıt atılmaz. Ara adımda hata olursa yazılanlar silinir (spec §4.4).
- **Cache-Control:** `public, max-age=31536000, immutable` — key'ler rastgele, içerik hiç değişmez (yeni fotoğraf = yeni key).
- **Yükleme senkron.** Kuyruk yok — tek fotoğraf birkaç yüz ms.
- **Maks dosya:** 10MB. Fiber `BodyLimit` Plan 1'de zaten ayarlı.
- **`ImageStore` Fiber'i import etmez.** Fiber sadece `internal/api/` altında.

---

## Dosya Yapısı

```
internal/image/
  store.go          → ImageStore interface, Size tipi, key üretimi
  local.go          → dosya sistemi implementasyonu (geliştirme/test)
  r2.go             → Cloudflare R2 implementasyonu (production)
  process.go        → decode → resize → WebP encode (saklamadan bağımsız)
  model.go          → ProductImage
  db.go             → product_images tablosu (store katmanı)
  service.go        → işleme + saklama + DB'yi birleştirir

internal/api/idare/
  image_handler.go  → upload, delete, reorder uçları
  image_view.go     → ImageView (URL'ler burada üretilir)

cmd/server/main.go  → ImageStore seçimi ve bağlama (modify)
pkg/config/config.go → R2 ayarları (modify)
```

**Neden `db.go` ayrı:** `store.go` saklama (R2/disk), `db.go` veritabanı. İkisi de "store" kelimesini hak ediyor ama farklı şeyler — isimlendirme çakışmasını dosya ayrımıyla çözüyoruz.

---

## Task 1: ImageStore interface ve local implementasyon

**Files:**
- Create: `internal/image/store.go`, `internal/image/local.go`
- Test: `internal/image/local_test.go`

**Interfaces:**
- Produces:
  - `image.Size` (string tipi), sabitler: `image.Size400 = "400"`, `image.Size1200 = "1200"`
  - `image.AllSizes = []Size{Size400, Size1200}`
  - `image.Store` interface: `Put(ctx, key string, size Size, data []byte) error`, `Delete(ctx, key string) error`, `URL(key string, size Size) string`
  - `image.NewKey() string` — rastgele, çakışmaz
  - `image.NewLocalStore(baseDir, baseURL string) (*LocalStore, error)`

- [ ] **Step 1: store.go yaz**

`internal/image/store.go`:
```go
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
```

- [ ] **Step 2: Testi yaz**

`internal/image/local_test.go`:
```go
package image

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLocalStore(t *testing.T) *LocalStore {
	t.Helper()
	store, err := NewLocalStore(t.TempDir(), "http://localhost:8080/uploads")
	require.NoError(t, err)
	return store
}

func TestNewKey_IsUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		k := NewKey()
		assert.False(t, seen[k], "key tekrar etti: %s", k)
		seen[k] = true
	}
}

func TestNewKey_Length(t *testing.T) {
	assert.Len(t, NewKey(), 16)
}

func TestSize_Width(t *testing.T) {
	assert.Equal(t, 400, Size400.Width())
	assert.Equal(t, 1200, Size1200.Width())
	assert.Equal(t, 0, Size("999").Width())
}

func TestSize_Valid(t *testing.T) {
	assert.True(t, Size400.Valid())
	assert.True(t, Size1200.Valid())
	assert.False(t, Size("999").Valid())
}

func TestLocalStore_PutAndRead(t *testing.T) {
	store := newTestLocalStore(t)
	ctx := context.Background()
	key := NewKey()

	err := store.Put(ctx, key, Size400, []byte("fake-webp-data"))

	require.NoError(t, err)
	path := filepath.Join(store.baseDir, "products", key, "400.webp")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "fake-webp-data", string(data))
}

func TestLocalStore_Put_CreatesNestedDirs(t *testing.T) {
	store := newTestLocalStore(t)

	err := store.Put(context.Background(), NewKey(), Size1200, []byte("x"))

	require.NoError(t, err)
}

func TestLocalStore_Put_Overwrites(t *testing.T) {
	store := newTestLocalStore(t)
	ctx := context.Background()
	key := NewKey()
	require.NoError(t, store.Put(ctx, key, Size400, []byte("eski")))

	require.NoError(t, store.Put(ctx, key, Size400, []byte("yeni")))

	data, err := os.ReadFile(filepath.Join(store.baseDir, "products", key, "400.webp"))
	require.NoError(t, err)
	assert.Equal(t, "yeni", string(data))
}

func TestLocalStore_Delete_RemovesAllSizes(t *testing.T) {
	store := newTestLocalStore(t)
	ctx := context.Background()
	key := NewKey()
	require.NoError(t, store.Put(ctx, key, Size400, []byte("a")))
	require.NoError(t, store.Put(ctx, key, Size1200, []byte("b")))

	require.NoError(t, store.Delete(ctx, key))

	_, err := os.Stat(filepath.Join(store.baseDir, "products", key))
	assert.True(t, os.IsNotExist(err), "key dizini tamamen silinmeli")
}

// Silme idempotent olmalı — olmayan key hata vermez.
// Sebep: ürün silme akışında DB kaydı gitti ama dosya yoksa, akış
// kırılmamalı (spec §4.4).
func TestLocalStore_Delete_MissingKeyIsNoError(t *testing.T) {
	store := newTestLocalStore(t)

	err := store.Delete(context.Background(), "olmayan-key")

	assert.NoError(t, err)
}

func TestLocalStore_URL(t *testing.T) {
	store := newTestLocalStore(t)

	url := store.URL("abc123", Size400)

	assert.Equal(t, "http://localhost:8080/uploads/products/abc123/400.webp", url)
}

func TestLocalStore_URL_TrimsTrailingSlash(t *testing.T) {
	store, err := NewLocalStore(t.TempDir(), "http://localhost:8080/uploads/")
	require.NoError(t, err)

	url := store.URL("abc123", Size1200)

	assert.Equal(t, "http://localhost:8080/uploads/products/abc123/1200.webp", url,
		"çift slash olmamalı")
}

// Path traversal koruması — key dışarıdan gelmese de savunma.
func TestLocalStore_Put_RejectsTraversalKey(t *testing.T) {
	store := newTestLocalStore(t)

	err := store.Put(context.Background(), "../../etc/passwd", Size400, []byte("x"))

	require.Error(t, err)
}
```

- [ ] **Step 3: Testi çalıştır, başarısız olduğunu gör**

Run: `go test ./internal/image/ -v`
Expected: FAIL — `undefined: NewLocalStore`

- [ ] **Step 4: local.go yaz**

`internal/image/local.go`:
```go
package image

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalStore görselleri dosya sistemine yazar. Geliştirme ve testte kullanılır.
// Production'da R2Store kullanılır — ikisi de Store interface'ini uygular,
// uygulama kodu hangisi olduğunu bilmez (spec §4.4).
type LocalStore struct {
	baseDir string
	baseURL string
}

func NewLocalStore(baseDir, baseURL string) (*LocalStore, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("baseDir boş olamaz")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("upload dizini oluştur: %w", err)
	}
	return &LocalStore{
		baseDir: baseDir,
		baseURL: strings.TrimRight(baseURL, "/"),
	}, nil
}

// safeKey key'in dizin dışına çıkmadığını doğrular.
func safeKey(key string) error {
	if key == "" {
		return fmt.Errorf("key boş olamaz")
	}
	if strings.Contains(key, "/") || strings.Contains(key, "\\") || strings.Contains(key, "..") {
		return fmt.Errorf("geçersiz key: %q", key)
	}
	return nil
}

func (s *LocalStore) Put(ctx context.Context, key string, size Size, data []byte) error {
	if err := safeKey(key); err != nil {
		return err
	}
	if !size.Valid() {
		return fmt.Errorf("geçersiz boyut: %q", size)
	}

	full := filepath.Join(s.baseDir, objectPath(key, size))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("dizin oluştur: %w", err)
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return fmt.Errorf("dosya yaz: %w", err)
	}
	return nil
}

// Delete key'in tüm boyutlarını siler. Olmayan key hata değil — idempotent.
func (s *LocalStore) Delete(ctx context.Context, key string) error {
	if err := safeKey(key); err != nil {
		return err
	}
	dir := filepath.Join(s.baseDir, "products", key)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("dosya sil: %w", err)
	}
	return nil
}

func (s *LocalStore) URL(key string, size Size) string {
	return s.baseURL + "/" + objectPath(key, size)
}
```

- [ ] **Step 5: Testi çalıştır**

Run: `go test ./internal/image/ -v`
Expected: PASS — 12 test

- [ ] **Step 6: Commit**

```bash
git add internal/image/store.go internal/image/local.go internal/image/local_test.go
git commit -m "feat: ImageStore interface ve local implementasyon"
```

---

## Task 2: Görsel işleme — decode, resize, WebP encode

**Files:**
- Create: `internal/image/process.go`
- Test: `internal/image/process_test.go`

**Interfaces:**
- Produces:
  - `image.ErrUnsupportedFormat` (sentinel hata)
  - `image.DetectFormat(data []byte) (string, error)` — "jpeg" | "png" | "webp"
  - `image.Process(data []byte, size Size) ([]byte, error)` — decode + resize + WebP encode

- [ ] **Step 1: Bağımlılıkları ekle**

```bash
go get github.com/disintegration/imaging
go get github.com/gen2brain/webp
go get golang.org/x/image/webp
```

`gen2brain/webp` saf Go (WASM tabanlı) — cgo yok. Seçim gerekçesi: encode yükleme başına bir kez oluyor, sonra CDN sonsuza dek statik servis ediyor. Yavaşlık nadir yola (esnafın upload'ı, birkaç yüz ms), cgo maliyeti ise her Docker build'ine yansırdı. Asimetri saf Go'yu net kazançlı yapıyor.

- [ ] **Step 2: Testi yaz**

`internal/image/process_test.go`:
```go
package image

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "golang.org/x/image/webp"
)

// makeJPEG test için verilen boyutta bir JPEG üretir.
func makeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: uint8(x % 256), B: uint8(y % 256), A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestDetectFormat_JPEG(t *testing.T) {
	format, err := DetectFormat(makeJPEG(t, 100, 100))

	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)
}

func TestDetectFormat_PNG(t *testing.T) {
	format, err := DetectFormat(makePNG(t, 100, 100))

	require.NoError(t, err)
	assert.Equal(t, "png", format)
}

func TestDetectFormat_WebP(t *testing.T) {
	webpData, err := Process(makeJPEG(t, 100, 100), Size400)
	require.NoError(t, err)

	format, err := DetectFormat(webpData)

	require.NoError(t, err)
	assert.Equal(t, "webp", format)
}

func TestDetectFormat_RejectsPDF(t *testing.T) {
	pdfHeader := []byte("%PDF-1.4\n%âãÏÓ\nrest of the file")

	_, err := DetectFormat(pdfHeader)

	require.ErrorIs(t, err, ErrUnsupportedFormat)
}

func TestDetectFormat_RejectsGIF(t *testing.T) {
	gifHeader := append([]byte("GIF89a"), bytes.Repeat([]byte{0}, 100)...)

	_, err := DetectFormat(gifHeader)

	require.ErrorIs(t, err, ErrUnsupportedFormat)
}

func TestDetectFormat_RejectsEmpty(t *testing.T) {
	_, err := DetectFormat([]byte{})

	require.ErrorIs(t, err, ErrUnsupportedFormat)
}

// Uzantı değil içerik belirler — .jpg uzantılı PDF reddedilmeli.
func TestDetectFormat_ContentNotExtension(t *testing.T) {
	fakeJPEG := []byte("%PDF-1.4 bu aslında bir PDF ama adı photo.jpg olabilir")

	_, err := DetectFormat(fakeJPEG)

	require.ErrorIs(t, err, ErrUnsupportedFormat)
}

func TestProcess_JPEGToWebP(t *testing.T) {
	out, err := Process(makeJPEG(t, 2000, 1500), Size400)

	require.NoError(t, err)
	require.NotEmpty(t, out)
	format, err := DetectFormat(out)
	require.NoError(t, err)
	assert.Equal(t, "webp", format, "çıktı her zaman WebP olmalı (spec §4.4)")
}

func TestProcess_PNGToWebP(t *testing.T) {
	out, err := Process(makePNG(t, 800, 600), Size1200)

	require.NoError(t, err)
	format, err := DetectFormat(out)
	require.NoError(t, err)
	assert.Equal(t, "webp", format)
}

func TestProcess_ResizesToTargetWidth(t *testing.T) {
	out, err := Process(makeJPEG(t, 2000, 1000), Size400)
	require.NoError(t, err)

	decoded, _, err := image.Decode(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, 400, decoded.Bounds().Dx())
}

func TestProcess_PreservesAspectRatio(t *testing.T) {
	// 2000x1000 → oran 2:1 → 400 genişlikte yükseklik 200 olmalı
	out, err := Process(makeJPEG(t, 2000, 1000), Size400)
	require.NoError(t, err)

	decoded, _, err := image.Decode(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, 400, decoded.Bounds().Dx())
	assert.Equal(t, 200, decoded.Bounds().Dy())
}

// Küçük görsel büyütülmez — 300px'lik fotoğraf 1200'e şişirilirse
// bulanıklaşır ve dosya boyutu boşuna büyür.
func TestProcess_DoesNotUpscale(t *testing.T) {
	out, err := Process(makeJPEG(t, 300, 200), Size1200)
	require.NoError(t, err)

	decoded, _, err := image.Decode(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, 300, decoded.Bounds().Dx(), "küçük görsel büyütülmemeli")
	assert.Equal(t, 200, decoded.Bounds().Dy())
}

func TestProcess_ExactTargetWidthUnchanged(t *testing.T) {
	out, err := Process(makeJPEG(t, 400, 300), Size400)
	require.NoError(t, err)

	decoded, _, err := image.Decode(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, 400, decoded.Bounds().Dx())
}

func TestProcess_RejectsUnsupportedFormat(t *testing.T) {
	_, err := Process([]byte("%PDF-1.4 bu bir PDF"), Size400)

	require.ErrorIs(t, err, ErrUnsupportedFormat)
}

func TestProcess_RejectsInvalidSize(t *testing.T) {
	_, err := Process(makeJPEG(t, 100, 100), Size("999"))

	require.Error(t, err)
}

func TestProcess_RejectsCorruptData(t *testing.T) {
	// Geçerli JPEG başlığı ama bozuk gövde
	corrupt := append(makeJPEG(t, 100, 100)[:20], bytes.Repeat([]byte{0xFF}, 50)...)

	_, err := Process(corrupt, Size400)

	require.Error(t, err)
}

// WebP çıktısı orijinalden küçük olmalı — sıkıştırmanın işe yaradığının kanıtı.
func TestProcess_OutputSmallerThanOriginal(t *testing.T) {
	original := makeJPEG(t, 2000, 1500)

	out, err := Process(original, Size400)

	require.NoError(t, err)
	assert.Less(t, len(out), len(original),
		"400px WebP, 2000px JPEG'den küçük olmalı")
}
```

- [ ] **Step 3: Testi çalıştır, başarısız olduğunu gör**

Run: `go test ./internal/image/ -run TestProcess -v`
Expected: FAIL — `undefined: Process`

- [ ] **Step 4: process.go yaz**

`internal/image/process.go`:
```go
package image

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"net/http"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"

	// Decoder kayıtları — image.Decode bunlar olmadan formatı tanımaz.
	_ "image/jpeg"
	_ "image/png"
	_ "golang.org/x/image/webp"
)

// ErrUnsupportedFormat desteklenmeyen girdi formatı.
var ErrUnsupportedFormat = errors.New("desteklenmeyen görsel formatı")

// webpQuality 1-100. 80 gözle fark edilmeyen kayıpla iyi sıkıştırma verir.
const webpQuality = 80

// DetectFormat görsel formatını İÇERİKTEN tespit eder, uzantıdan değil.
// Uzantı yalan söyleyebilir: .jpg uzantılı bir dosya PDF olabilir.
// Kabul edilen girdiler: jpeg, png, webp (spec §4.4 — girdi serbest, çıktı WebP).
func DetectFormat(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("%w: boş dosya", ErrUnsupportedFormat)
	}

	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/jpeg":
		return "jpeg", nil
	case "image/png":
		return "png", nil
	case "image/webp":
		return "webp", nil
	default:
		return "", fmt.Errorf("%w: %s (sadece JPEG, PNG veya WebP)",
			ErrUnsupportedFormat, contentType)
	}
}

// Process görseli decode eder, hedef genişliğe küçültür ve WebP olarak encode eder.
// Küçük görseller büyütülmez — bulanıklaşır ve dosya boşuna büyür.
// En-boy oranı korunur.
func Process(data []byte, size Size) ([]byte, error) {
	if !size.Valid() {
		return nil, fmt.Errorf("geçersiz boyut: %q", size)
	}

	if _, err := DetectFormat(data); err != nil {
		return nil, err
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("görsel decode: %w", err)
	}

	resized := resizeToWidth(src, size.Width())

	var buf bytes.Buffer
	if err := webp.Encode(&buf, resized, webp.Options{Quality: webpQuality}); err != nil {
		return nil, fmt.Errorf("webp encode: %w", err)
	}

	return buf.Bytes(), nil
}

// resizeToWidth en-boy oranını koruyarak hedef genişliğe küçültür.
// Görsel zaten hedeften darsa dokunmaz.
func resizeToWidth(src image.Image, targetWidth int) image.Image {
	if src.Bounds().Dx() <= targetWidth {
		return src
	}
	// imaging.Resize'da height=0 → oran korunarak otomatik hesaplanır.
	return imaging.Resize(src, targetWidth, 0, imaging.Lanczos)
}
```

- [ ] **Step 5: Testi çalıştır**

Run: `go test ./internal/image/ -v`
Expected: PASS — tüm process ve local testleri

- [ ] **Step 6: Commit**

```bash
git add internal/image/process.go internal/image/process_test.go go.mod go.sum
git commit -m "feat: görsel işleme — JPEG/PNG/WebP girdi, WebP çıktı, resize"
```

---

## Task 3: product_images DB katmanı

**Files:**
- Create: `internal/image/model.go`, `internal/image/db.go`
- Test: `internal/image/db_test.go`

**Interfaces:**
- Consumes: `database.NewTestDB`, `errorsx`
- Produces:
  - `image.ProductImage{ID int64, ProductID int64, ImageKey string, SortOrder int}`
  - `image.NewDB(pool *pgxpool.Pool) *DB`
  - `(*DB).Insert(ctx, productID int64, key string) (*ProductImage, error)` — sort_order otomatik son sıra
  - `(*DB).ListByProduct(ctx, productID int64) ([]ProductImage, error)`
  - `(*DB).ListByProducts(ctx, productIDs []int64) (map[int64][]ProductImage, error)`
  - `(*DB).GetByID(ctx, id int64) (*ProductImage, error)`
  - `(*DB).Delete(ctx, id int64) error`
  - `(*DB).KeysByProduct(ctx, productID int64) ([]string, error)`
  - `(*DB).Reorder(ctx, productID int64, imageIDs []int64) error`

- [ ] **Step 1: model.go yaz**

`internal/image/model.go`:
```go
package image

// ProductImage bir ürünün görseli. sort_order=0 olan kapak (spec §4.4).
type ProductImage struct {
	ID        int64
	ProductID int64
	ImageKey  string
	SortOrder int
}
```

- [ ] **Step 2: Testi yaz**

`internal/image/db_test.go`:
```go
package image

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) (*DB, *pgxpool.Pool, context.Context) {
	t.Helper()
	pool := database.NewTestDB(t)
	return NewDB(pool), pool, context.Background()
}

// insertProduct test için doğrudan ürün ekler — product paketine bağımlılık
// yaratmamak için (import cycle olurdu).
func insertProduct(t *testing.T, pool *pgxpool.Pool, name string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO products (name, price, is_active) VALUES ($1, 100, true) RETURNING id`,
		name,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func TestDB_Insert_FirstImageIsSortZero(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pid := insertProduct(t, pool, "Buket")

	img, err := db.Insert(ctx, pid, "key-1")

	require.NoError(t, err)
	assert.Equal(t, 0, img.SortOrder, "ilk görsel kapak olmalı (sort_order=0)")
	assert.Equal(t, "key-1", img.ImageKey)
	assert.Equal(t, pid, img.ProductID)
}

func TestDB_Insert_AppendsToEnd(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pid := insertProduct(t, pool, "Buket")

	first, err := db.Insert(ctx, pid, "key-1")
	require.NoError(t, err)
	second, err := db.Insert(ctx, pid, "key-2")
	require.NoError(t, err)
	third, err := db.Insert(ctx, pid, "key-3")
	require.NoError(t, err)

	assert.Equal(t, 0, first.SortOrder)
	assert.Equal(t, 1, second.SortOrder)
	assert.Equal(t, 2, third.SortOrder)
}

func TestDB_Insert_SortOrderIsPerProduct(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pidA := insertProduct(t, pool, "A")
	pidB := insertProduct(t, pool, "B")

	_, err := db.Insert(ctx, pidA, "a-1")
	require.NoError(t, err)
	bFirst, err := db.Insert(ctx, pidB, "b-1")
	require.NoError(t, err)

	assert.Equal(t, 0, bFirst.SortOrder, "her ürünün sıralaması bağımsız")
}

func TestDB_ListByProduct_OrderedBySortOrder(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pid := insertProduct(t, pool, "Buket")
	for _, k := range []string{"key-1", "key-2", "key-3"} {
		_, err := db.Insert(ctx, pid, k)
		require.NoError(t, err)
	}

	list, err := db.ListByProduct(ctx, pid)

	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, "key-1", list[0].ImageKey)
	assert.Equal(t, "key-2", list[1].ImageKey)
	assert.Equal(t, "key-3", list[2].ImageKey)
}

func TestDB_ListByProduct_Empty(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pid := insertProduct(t, pool, "Görselsiz")

	list, err := db.ListByProduct(ctx, pid)

	require.NoError(t, err)
	assert.Empty(t, list)
}

// ListByProducts liste sayfası için — N+1 sorgu önlenir.
func TestDB_ListByProducts_GroupsByProductID(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pidA := insertProduct(t, pool, "A")
	pidB := insertProduct(t, pool, "B")
	_, err := db.Insert(ctx, pidA, "a-1")
	require.NoError(t, err)
	_, err = db.Insert(ctx, pidA, "a-2")
	require.NoError(t, err)
	_, err = db.Insert(ctx, pidB, "b-1")
	require.NoError(t, err)

	grouped, err := db.ListByProducts(ctx, []int64{pidA, pidB})

	require.NoError(t, err)
	require.Len(t, grouped[pidA], 2)
	require.Len(t, grouped[pidB], 1)
	assert.Equal(t, "a-1", grouped[pidA][0].ImageKey)
}

func TestDB_ListByProducts_EmptyInput(t *testing.T) {
	db, _, ctx := newTestDB(t)

	grouped, err := db.ListByProducts(ctx, []int64{})

	require.NoError(t, err)
	assert.Empty(t, grouped)
}

func TestDB_GetByID(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pid := insertProduct(t, pool, "Buket")
	inserted, err := db.Insert(ctx, pid, "key-1")
	require.NoError(t, err)

	fetched, err := db.GetByID(ctx, inserted.ID)

	require.NoError(t, err)
	assert.Equal(t, "key-1", fetched.ImageKey)
}

func TestDB_GetByID_NotFound(t *testing.T) {
	db, _, ctx := newTestDB(t)

	_, err := db.GetByID(ctx, 9999)

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestDB_Delete(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pid := insertProduct(t, pool, "Buket")
	img, err := db.Insert(ctx, pid, "key-1")
	require.NoError(t, err)

	require.NoError(t, db.Delete(ctx, img.ID))

	_, err = db.GetByID(ctx, img.ID)
	assert.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestDB_Delete_NotFound(t *testing.T) {
	db, _, ctx := newTestDB(t)

	err := db.Delete(ctx, 9999)

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestDB_KeysByProduct(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pid := insertProduct(t, pool, "Buket")
	_, err := db.Insert(ctx, pid, "key-1")
	require.NoError(t, err)
	_, err = db.Insert(ctx, pid, "key-2")
	require.NoError(t, err)

	keys, err := db.KeysByProduct(ctx, pid)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"key-1", "key-2"}, keys)
}

func TestDB_Reorder(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pid := insertProduct(t, pool, "Buket")
	first, err := db.Insert(ctx, pid, "key-1")
	require.NoError(t, err)
	second, err := db.Insert(ctx, pid, "key-2")
	require.NoError(t, err)
	third, err := db.Insert(ctx, pid, "key-3")
	require.NoError(t, err)

	// Ters çevir: 3, 2, 1
	require.NoError(t, db.Reorder(ctx, pid, []int64{third.ID, second.ID, first.ID}))

	list, err := db.ListByProduct(ctx, pid)
	require.NoError(t, err)
	assert.Equal(t, "key-3", list[0].ImageKey, "yeni kapak key-3 olmalı")
	assert.Equal(t, "key-2", list[1].ImageKey)
	assert.Equal(t, "key-1", list[2].ImageKey)
}

// Başka ürünün görseli sıralamaya sokulamaz.
func TestDB_Reorder_RejectsForeignImage(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pidA := insertProduct(t, pool, "A")
	pidB := insertProduct(t, pool, "B")
	aImg, err := db.Insert(ctx, pidA, "a-1")
	require.NoError(t, err)
	bImg, err := db.Insert(ctx, pidB, "b-1")
	require.NoError(t, err)

	err = db.Reorder(ctx, pidA, []int64{aImg.ID, bImg.ID})

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

// Eksik id ile sıralama reddedilir — sessizce yarım sıralama olmasın.
func TestDB_Reorder_RejectsIncompleteList(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pid := insertProduct(t, pool, "Buket")
	first, err := db.Insert(ctx, pid, "key-1")
	require.NoError(t, err)
	_, err = db.Insert(ctx, pid, "key-2")
	require.NoError(t, err)

	err = db.Reorder(ctx, pid, []int64{first.ID})

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

// Ürün silinince görsel kayıtları CASCADE ile gider (spec §4.4).
func TestDB_ProductDelete_CascadesImages(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pid := insertProduct(t, pool, "Silinecek")
	img, err := db.Insert(ctx, pid, "key-1")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, pid)
	require.NoError(t, err)

	_, err = db.GetByID(ctx, img.ID)
	assert.ErrorIs(t, err, errorsx.ErrNotFound)
}
```

- [ ] **Step 3: Testi çalıştır, başarısız olduğunu gör**

Run: `go test ./internal/image/ -run TestDB -v`
Expected: FAIL — `undefined: NewDB`

- [ ] **Step 4: db.go yaz**

`internal/image/db.go`:
```go
package image

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// DB product_images tablosunu yönetir. Saklama (R2/disk) ile karıştırma —
// o Store interface'inin işi.
type DB struct {
	pool *pgxpool.Pool
}

func NewDB(pool *pgxpool.Pool) *DB {
	return &DB{pool: pool}
}

// Insert görseli ürünün sonuna ekler. İlk görsel sort_order=0 → kapak.
func (d *DB) Insert(ctx context.Context, productID int64, key string) (*ProductImage, error) {
	var img ProductImage
	err := d.pool.QueryRow(ctx,
		`INSERT INTO product_images (product_id, image_key, sort_order)
		 VALUES ($1, $2, COALESCE(
		     (SELECT max(sort_order) + 1 FROM product_images WHERE product_id = $1),
		     0
		 ))
		 RETURNING id, product_id, image_key, sort_order`,
		productID, key,
	).Scan(&img.ID, &img.ProductID, &img.ImageKey, &img.SortOrder)

	if err != nil {
		return nil, fmt.Errorf("görsel ekle: %w", err)
	}
	return &img, nil
}

func (d *DB) ListByProduct(ctx context.Context, productID int64) ([]ProductImage, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT id, product_id, image_key, sort_order
		 FROM product_images WHERE product_id = $1
		 ORDER BY sort_order, id`,
		productID)
	if err != nil {
		return nil, fmt.Errorf("görsel listele: %w", err)
	}
	defer rows.Close()

	return scanImages(rows)
}

// ListByProducts liste sayfası için — tek sorguda tüm ürünlerin görselleri.
// N+1 sorgu problemini önler.
func (d *DB) ListByProducts(ctx context.Context, productIDs []int64) (map[int64][]ProductImage, error) {
	out := make(map[int64][]ProductImage)
	if len(productIDs) == 0 {
		return out, nil
	}

	rows, err := d.pool.Query(ctx,
		`SELECT id, product_id, image_key, sort_order
		 FROM product_images WHERE product_id = ANY($1)
		 ORDER BY product_id, sort_order, id`,
		productIDs)
	if err != nil {
		return nil, fmt.Errorf("görseller listele: %w", err)
	}
	defer rows.Close()

	list, err := scanImages(rows)
	if err != nil {
		return nil, err
	}

	for _, img := range list {
		out[img.ProductID] = append(out[img.ProductID], img)
	}
	return out, nil
}

func scanImages(rows pgx.Rows) ([]ProductImage, error) {
	out := make([]ProductImage, 0)
	for rows.Next() {
		var img ProductImage
		if err := rows.Scan(&img.ID, &img.ProductID, &img.ImageKey, &img.SortOrder); err != nil {
			return nil, fmt.Errorf("görsel scan: %w", err)
		}
		out = append(out, img)
	}
	return out, rows.Err()
}

func (d *DB) GetByID(ctx context.Context, id int64) (*ProductImage, error) {
	var img ProductImage
	err := d.pool.QueryRow(ctx,
		`SELECT id, product_id, image_key, sort_order FROM product_images WHERE id = $1`,
		id,
	).Scan(&img.ID, &img.ProductID, &img.ImageKey, &img.SortOrder)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("görsel ara: %w", err)
	}
	return &img, nil
}

func (d *DB) Delete(ctx context.Context, id int64) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM product_images WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("görsel sil: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

// KeysByProduct ürün silinmeden ÖNCE çağrılır — saklamadan temizlemek için
// key'ler lazım, DB kaydı gidince öğrenilemez (spec §4.4).
func (d *DB) KeysByProduct(ctx context.Context, productID int64) ([]string, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT image_key FROM product_images WHERE product_id = $1`, productID)
	if err != nil {
		return nil, fmt.Errorf("key listele: %w", err)
	}
	defer rows.Close()

	keys := make([]string, 0)
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("key scan: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// Reorder görselleri verilen sıraya dizer. imageIDs ürünün TÜM görsellerini
// içermeli — eksik veya yabancı id reddedilir.
func (d *DB) Reorder(ctx context.Context, productID int64, imageIDs []int64) error {
	current, err := d.ListByProduct(ctx, productID)
	if err != nil {
		return err
	}

	if len(current) != len(imageIDs) {
		return fmt.Errorf("%w: %d görsel bekleniyordu, %d geldi",
			errorsx.ErrInvalidInput, len(current), len(imageIDs))
	}

	owned := make(map[int64]bool, len(current))
	for _, img := range current {
		owned[img.ID] = true
	}
	seen := make(map[int64]bool, len(imageIDs))
	for _, id := range imageIDs {
		if !owned[id] {
			return fmt.Errorf("%w: görsel %d bu ürüne ait değil", errorsx.ErrInvalidInput, id)
		}
		if seen[id] {
			return fmt.Errorf("%w: görsel %d tekrar etti", errorsx.ErrInvalidInput, id)
		}
		seen[id] = true
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	for i, id := range imageIDs {
		if _, err := tx.Exec(ctx,
			`UPDATE product_images SET sort_order = $1 WHERE id = $2`, i, id); err != nil {
			return fmt.Errorf("sıra güncelle: %w", err)
		}
	}

	return tx.Commit(ctx)
}
```

- [ ] **Step 5: Testi çalıştır**

Run: `go test ./internal/image/ -v`
Expected: PASS — tüm testler

- [ ] **Step 6: Commit**

```bash
git add internal/image/model.go internal/image/db.go internal/image/db_test.go
git commit -m "feat: product_images DB katmanı — sıralama, gruplama"
```

---

## Task 4: Image service — işleme + saklama + DB

Spec §4.4'ün atomiklik kuralının uygulandığı yer.

**Files:**
- Create: `internal/image/service.go`
- Test: `internal/image/service_test.go`

**Interfaces:**
- Consumes: `image.Store`, `image.DB`, `image.Process`
- Produces:
  - `image.NewService(store Store, db *DB) *Service`
  - `(*Service).Upload(ctx, productID int64, data []byte) (*ProductImage, error)`
  - `(*Service).Delete(ctx, imageID int64) error`
  - `(*Service).DeleteAllForProduct(ctx, productID int64) error`
  - `(*Service).ListByProduct(ctx, productID int64) ([]ProductImage, error)`
  - `(*Service).ListByProducts(ctx, productIDs []int64) (map[int64][]ProductImage, error)`
  - `(*Service).Reorder(ctx, productID int64, imageIDs []int64) error`
  - `(*Service).URL(key string, size Size) string`

- [ ] **Step 1: Testi yaz**

`internal/image/service_test.go`:
```go
package image

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStore Store interface'ini taklit eder — hangi çağrının yapıldığını
// kaydeder ve istenen çağrıda hata döndürebilir. Atomiklik testleri için.
type fakeStore struct {
	mu       sync.Mutex
	put      map[string][]byte
	deleted  []string
	failPutN int // 0 = hata yok, N = N'inci Put çağrısı hata verir
	putCount int
}

func newFakeStore() *fakeStore {
	return &fakeStore{put: make(map[string][]byte)}
}

func (f *fakeStore) Put(ctx context.Context, key string, size Size, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.putCount++
	if f.failPutN > 0 && f.putCount == f.failPutN {
		return errors.New("simüle edilmiş saklama hatası")
	}
	f.put[objectPath(key, size)] = data
	return nil
}

func (f *fakeStore) Delete(ctx context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = append(f.deleted, key)
	for _, size := range AllSizes {
		delete(f.put, objectPath(key, size))
	}
	return nil
}

func (f *fakeStore) URL(key string, size Size) string {
	return "https://cdn.test/" + objectPath(key, size)
}

func (f *fakeStore) storedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.put)
}

func newTestService(t *testing.T) (*Service, *fakeStore, *pgxpool.Pool, context.Context) {
	t.Helper()
	pool := database.NewTestDB(t)
	store := newFakeStore()
	return NewService(store, NewDB(pool)), store, pool, context.Background()
}

func TestService_Upload_StoresBothSizes(t *testing.T) {
	svc, store, pool, ctx := newTestService(t)
	pid := insertProduct(t, pool, "Buket")

	img, err := svc.Upload(ctx, pid, makeJPEG(t, 2000, 1500))

	require.NoError(t, err)
	assert.Equal(t, 2, store.storedCount(), "400 ve 1200 yazılmalı")
	assert.NotEmpty(t, img.ImageKey)
	assert.Equal(t, 0, img.SortOrder)
}

func TestService_Upload_StoredDataIsWebP(t *testing.T) {
	svc, store, pool, ctx := newTestService(t)
	pid := insertProduct(t, pool, "Buket")

	img, err := svc.Upload(ctx, pid, makeJPEG(t, 800, 600))
	require.NoError(t, err)

	stored := store.put[objectPath(img.ImageKey, Size400)]
	require.NotEmpty(t, stored)
	format, err := DetectFormat(stored)
	require.NoError(t, err)
	assert.Equal(t, "webp", format)
}

func TestService_Upload_AcceptsPNG(t *testing.T) {
	svc, _, pool, ctx := newTestService(t)
	pid := insertProduct(t, pool, "Buket")

	_, err := svc.Upload(ctx, pid, makePNG(t, 600, 400))

	require.NoError(t, err)
}

func TestService_Upload_RejectsUnsupportedFormat(t *testing.T) {
	svc, store, pool, ctx := newTestService(t)
	pid := insertProduct(t, pool, "Buket")

	_, err := svc.Upload(ctx, pid, []byte("%PDF-1.4 bu bir PDF"))

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
	assert.Equal(t, 0, store.storedCount(), "geçersiz format hiç yazılmamalı")
}

// Spec §4.4 atomiklik: ikinci boyut yazılamazsa birincisi geri alınır ve
// DB'ye kayıt atılmaz. Yarım görsel kalmaz.
func TestService_Upload_SecondSizeFails_RollsBackFirst(t *testing.T) {
	svc, store, pool, ctx := newTestService(t)
	pid := insertProduct(t, pool, "Buket")
	store.failPutN = 2 // ikinci Put patlasın

	_, err := svc.Upload(ctx, pid, makeJPEG(t, 1000, 800))

	require.Error(t, err)
	assert.Equal(t, 0, store.storedCount(), "yazılan ilk boyut temizlenmeli")
	assert.Len(t, store.deleted, 1, "temizlik için Delete çağrılmalı")

	imgs, err := NewDB(pool).ListByProduct(ctx, pid)
	require.NoError(t, err)
	assert.Empty(t, imgs, "DB'ye kayıt atılmamalı")
}

func TestService_Upload_FirstSizeFails_NoDBRecord(t *testing.T) {
	svc, store, pool, ctx := newTestService(t)
	pid := insertProduct(t, pool, "Buket")
	store.failPutN = 1

	_, err := svc.Upload(ctx, pid, makeJPEG(t, 1000, 800))

	require.Error(t, err)
	imgs, err := NewDB(pool).ListByProduct(ctx, pid)
	require.NoError(t, err)
	assert.Empty(t, imgs)
}

func TestService_Upload_MultipleImagesOrdered(t *testing.T) {
	svc, _, pool, ctx := newTestService(t)
	pid := insertProduct(t, pool, "Buket")

	first, err := svc.Upload(ctx, pid, makeJPEG(t, 800, 600))
	require.NoError(t, err)
	second, err := svc.Upload(ctx, pid, makeJPEG(t, 800, 600))
	require.NoError(t, err)

	assert.Equal(t, 0, first.SortOrder)
	assert.Equal(t, 1, second.SortOrder)
	assert.NotEqual(t, first.ImageKey, second.ImageKey, "her görselin key'i benzersiz")
}

func TestService_Delete_RemovesFromStoreAndDB(t *testing.T) {
	svc, store, pool, ctx := newTestService(t)
	pid := insertProduct(t, pool, "Buket")
	img, err := svc.Upload(ctx, pid, makeJPEG(t, 800, 600))
	require.NoError(t, err)

	require.NoError(t, svc.Delete(ctx, img.ID))

	assert.Equal(t, 0, store.storedCount())
	assert.Contains(t, store.deleted, img.ImageKey)
	_, err = NewDB(pool).GetByID(ctx, img.ID)
	assert.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestService_Delete_NotFound(t *testing.T) {
	svc, _, _, ctx := newTestService(t)

	err := svc.Delete(ctx, 9999)

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

// Ürün silme akışı: key'ler önce okunur, DB gider (CASCADE), sonra saklama
// temizlenir (spec §4.4).
func TestService_DeleteAllForProduct(t *testing.T) {
	svc, store, pool, ctx := newTestService(t)
	pid := insertProduct(t, pool, "Buket")
	_, err := svc.Upload(ctx, pid, makeJPEG(t, 800, 600))
	require.NoError(t, err)
	_, err = svc.Upload(ctx, pid, makeJPEG(t, 800, 600))
	require.NoError(t, err)

	require.NoError(t, svc.DeleteAllForProduct(ctx, pid))

	assert.Equal(t, 0, store.storedCount(), "tüm dosyalar silinmeli")
	assert.Len(t, store.deleted, 2)
}

func TestService_DeleteAllForProduct_NoImages(t *testing.T) {
	svc, _, pool, ctx := newTestService(t)
	pid := insertProduct(t, pool, "Görselsiz")

	err := svc.DeleteAllForProduct(ctx, pid)

	assert.NoError(t, err, "görseli olmayan ürün hata vermemeli")
}

func TestService_Reorder(t *testing.T) {
	svc, _, pool, ctx := newTestService(t)
	pid := insertProduct(t, pool, "Buket")
	first, err := svc.Upload(ctx, pid, makeJPEG(t, 800, 600))
	require.NoError(t, err)
	second, err := svc.Upload(ctx, pid, makeJPEG(t, 800, 600))
	require.NoError(t, err)

	require.NoError(t, svc.Reorder(ctx, pid, []int64{second.ID, first.ID}))

	list, err := svc.ListByProduct(ctx, pid)
	require.NoError(t, err)
	assert.Equal(t, second.ID, list[0].ID, "kapak değişmeli")
}

func TestService_URL(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	url := svc.URL("abc123", Size400)

	assert.Equal(t, "https://cdn.test/products/abc123/400.webp", url)
}
```

- [ ] **Step 2: Testi çalıştır, başarısız olduğunu gör**

Run: `go test ./internal/image/ -run TestService -v`
Expected: FAIL — `undefined: NewService`

- [ ] **Step 3: service.go yaz**

`internal/image/service.go`:
```go
package image

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// Service görsel işleme, saklama ve veritabanını birleştirir.
type Service struct {
	store Store
	db    *DB
}

func NewService(store Store, db *DB) *Service {
	return &Service{store: store, db: db}
}

// Upload görseli işler, tüm boyutları saklar ve DB'ye kaydeder.
//
// Atomiklik (spec §4.4): önce TÜM boyutlar saklamaya yazılır; hepsi
// başarılıysa DB'ye kayıt atılır. Ara adımda hata olursa yazılanlar silinir.
// Böylece DB kaydı olmayan dosya (yetim) veya dosyası olmayan DB kaydı
// (kırık görsel) oluşmaz.
func (s *Service) Upload(ctx context.Context, productID int64, data []byte) (*ProductImage, error) {
	processed := make(map[Size][]byte, len(AllSizes))
	for _, size := range AllSizes {
		out, err := Process(data, size)
		if err != nil {
			if errors.Is(err, ErrUnsupportedFormat) {
				return nil, fmt.Errorf("%w: %s", errorsx.ErrInvalidInput, err.Error())
			}
			return nil, fmt.Errorf("görsel işle (%s): %w", size, err)
		}
		processed[size] = out
	}

	key := NewKey()

	for _, size := range AllSizes {
		if err := s.store.Put(ctx, key, size, processed[size]); err != nil {
			// Yarım kalanı temizle — yetim dosya bırakma.
			if delErr := s.store.Delete(ctx, key); delErr != nil {
				log.Printf("yarım yükleme temizlenemedi (key=%s): %v", key, delErr)
			}
			return nil, fmt.Errorf("görsel sakla (%s): %w", size, err)
		}
	}

	img, err := s.db.Insert(ctx, productID, key)
	if err != nil {
		// DB yazılamadıysa dosyalar yetim kalır — temizle.
		if delErr := s.store.Delete(ctx, key); delErr != nil {
			log.Printf("DB hatası sonrası temizlik başarısız (key=%s): %v", key, delErr)
		}
		return nil, err
	}

	return img, nil
}

// Delete görseli DB'den ve saklamadan siler.
// Sıra önemli: key önce okunur (DB kaydı gidince öğrenilemez).
func (s *Service) Delete(ctx context.Context, imageID int64) error {
	img, err := s.db.GetByID(ctx, imageID)
	if err != nil {
		return err
	}

	if err := s.db.Delete(ctx, imageID); err != nil {
		return err
	}

	// Saklama silme başarısız olursa yetim dosya kalır ama site bozulmaz —
	// log'a düşer, kabul edilebilir (spec §4.4).
	if err := s.store.Delete(ctx, img.ImageKey); err != nil {
		log.Printf("yetim dosya: saklamadan silinemedi (key=%s): %v", img.ImageKey, err)
	}

	return nil
}

// DeleteAllForProduct ürün silinmeden ÖNCE çağrılır.
// Ürün silinince product_images CASCADE ile gider ve key'ler kaybolur —
// o yüzden dosyaları temizlemek için key'leri şimdi okumak gerekir.
func (s *Service) DeleteAllForProduct(ctx context.Context, productID int64) error {
	keys, err := s.db.KeysByProduct(ctx, productID)
	if err != nil {
		return err
	}

	for _, key := range keys {
		if err := s.store.Delete(ctx, key); err != nil {
			log.Printf("yetim dosya: saklamadan silinemedi (key=%s): %v", key, err)
		}
	}

	return nil
}

func (s *Service) ListByProduct(ctx context.Context, productID int64) ([]ProductImage, error) {
	return s.db.ListByProduct(ctx, productID)
}

func (s *Service) ListByProducts(ctx context.Context, productIDs []int64) (map[int64][]ProductImage, error) {
	return s.db.ListByProducts(ctx, productIDs)
}

func (s *Service) Reorder(ctx context.Context, productID int64, imageIDs []int64) error {
	return s.db.Reorder(ctx, productID, imageIDs)
}

// URL saklamanın ürettiği public adresi döner. Domain veritabanında değil,
// ImageStore'da yaşar (spec §4.1).
func (s *Service) URL(key string, size Size) string {
	return s.store.URL(key, size)
}
```

- [ ] **Step 4: Testi çalıştır**

Run: `go test ./internal/image/ -v`
Expected: PASS — tüm testler

- [ ] **Step 5: Commit**

```bash
git add internal/image/service.go internal/image/service_test.go
git commit -m "feat: image service — atomik yükleme, temizlik garantisi"
```

---

## Task 5: R2 implementasyonu

**Files:**
- Create: `internal/image/r2.go`
- Modify: `pkg/config/config.go` — R2 ayarları
- Modify: `.env.example`

**Interfaces:**
- Produces: `image.NewR2Store(cfg R2Config) (*R2Store, error)`, `image.R2Config{AccountID, AccessKeyID, SecretAccessKey, Bucket, PublicURL string}`

- [ ] **Step 1: AWS SDK ekle**

```bash
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/credentials
go get github.com/aws/aws-sdk-go-v2/service/s3
```

R2, S3-uyumlu API sunuyor — ayrı bir SDK yok, endpoint override yeterli.

- [ ] **Step 2: r2.go yaz**

`internal/image/r2.go`:
```go
package image

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// cacheControl key'ler rastgele ve içerik hiç değişmez (yeni fotoğraf =
// yeni key), bu yüzden immutable + 1 yıl. Tarayıcı ve CDN sonsuza dek
// cache'leyebilir.
const cacheControl = "public, max-age=31536000, immutable"

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicURL       string // R2 public bucket URL'i veya özel domain
}

// R2Store Cloudflare R2'ye yazar. S3-uyumlu API kullanır.
type R2Store struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewR2Store(cfg R2Config) (*R2Store, error) {
	if cfg.AccountID == "" || cfg.AccessKeyID == "" ||
		cfg.SecretAccessKey == "" || cfg.Bucket == "" || cfg.PublicURL == "" {
		return nil, fmt.Errorf("R2 ayarları eksik (account_id, access_key, secret, bucket, public_url gerekli)")
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &R2Store{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: strings.TrimRight(cfg.PublicURL, "/"),
	}, nil
}

func (s *R2Store) Put(ctx context.Context, key string, size Size, data []byte) error {
	if err := safeKey(key); err != nil {
		return err
	}
	if !size.Valid() {
		return fmt.Errorf("geçersiz boyut: %q", size)
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(s.bucket),
		Key:          aws.String(objectPath(key, size)),
		Body:         bytes.NewReader(data),
		ContentType:  aws.String("image/webp"),
		CacheControl: aws.String(cacheControl),
	})
	if err != nil {
		return fmt.Errorf("r2 put (%s/%s): %w", key, size, err)
	}
	return nil
}

// Delete key'in tüm boyutlarını siler. Olmayan obje hata değil — S3
// DeleteObject zaten idempotent.
func (s *R2Store) Delete(ctx context.Context, key string) error {
	if err := safeKey(key); err != nil {
		return err
	}

	for _, size := range AllSizes {
		_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(objectPath(key, size)),
		})
		if err != nil {
			return fmt.Errorf("r2 delete (%s/%s): %w", key, size, err)
		}
	}
	return nil
}

func (s *R2Store) URL(key string, size Size) string {
	return s.publicURL + "/" + objectPath(key, size)
}
```

- [ ] **Step 3: config.go'yu güncelle**

`pkg/config/config.go` — `Config` struct'ına ekle:
```go
type Config struct {
	DatabaseURL    string
	JWTSecret      string
	Port           string
	WhatsAppNumber string
	SiteURL        string

	// Görsel saklama. StorageDriver "local" veya "r2".
	StorageDriver string
	UploadDir     string // local için
	UploadBaseURL string // local için
	R2AccountID   string
	R2AccessKey   string
	R2SecretKey   string
	R2Bucket      string
	R2PublicURL   string
}
```

`Load()` içinde, mevcut doğrulamalardan sonra:
```go
	cfg.StorageDriver = os.Getenv("STORAGE_DRIVER")
	if cfg.StorageDriver == "" {
		cfg.StorageDriver = "local"
	}
	cfg.UploadDir = os.Getenv("UPLOAD_DIR")
	if cfg.UploadDir == "" {
		cfg.UploadDir = "./uploads"
	}
	cfg.UploadBaseURL = os.Getenv("UPLOAD_BASE_URL")
	if cfg.UploadBaseURL == "" {
		cfg.UploadBaseURL = "http://localhost:" + cfg.Port + "/uploads"
	}
	cfg.R2AccountID = os.Getenv("R2_ACCOUNT_ID")
	cfg.R2AccessKey = os.Getenv("R2_ACCESS_KEY_ID")
	cfg.R2SecretKey = os.Getenv("R2_SECRET_ACCESS_KEY")
	cfg.R2Bucket = os.Getenv("R2_BUCKET")
	cfg.R2PublicURL = os.Getenv("R2_PUBLIC_URL")

	switch cfg.StorageDriver {
	case "local":
	case "r2":
		if cfg.R2AccountID == "" || cfg.R2AccessKey == "" ||
			cfg.R2SecretKey == "" || cfg.R2Bucket == "" || cfg.R2PublicURL == "" {
			return nil, fmt.Errorf("STORAGE_DRIVER=r2 için R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_BUCKET, R2_PUBLIC_URL zorunlu")
		}
	default:
		return nil, fmt.Errorf("geçersiz STORAGE_DRIVER: %q (local veya r2)", cfg.StorageDriver)
	}
```

- [ ] **Step 4: Config testini genişlet**

`pkg/config/config_test.go` dosyasına ekle:
```go
func TestLoad_DefaultsToLocalStorage(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")
	t.Setenv("WHATSAPP_NUMBER", "905551234567")
	t.Setenv("SITE_URL", "https://example.com")
	t.Setenv("STORAGE_DRIVER", "")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "local", cfg.StorageDriver)
	assert.Equal(t, "./uploads", cfg.UploadDir)
}

func TestLoad_R2RequiresAllSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")
	t.Setenv("WHATSAPP_NUMBER", "905551234567")
	t.Setenv("SITE_URL", "https://example.com")
	t.Setenv("STORAGE_DRIVER", "r2")
	t.Setenv("R2_ACCOUNT_ID", "abc")
	t.Setenv("R2_ACCESS_KEY_ID", "")

	_, err := Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "R2_ACCESS_KEY_ID")
}

func TestLoad_R2WithAllSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")
	t.Setenv("WHATSAPP_NUMBER", "905551234567")
	t.Setenv("SITE_URL", "https://example.com")
	t.Setenv("STORAGE_DRIVER", "r2")
	t.Setenv("R2_ACCOUNT_ID", "abc")
	t.Setenv("R2_ACCESS_KEY_ID", "key")
	t.Setenv("R2_SECRET_ACCESS_KEY", "secret")
	t.Setenv("R2_BUCKET", "cicekci")
	t.Setenv("R2_PUBLIC_URL", "https://cdn.example.com")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "r2", cfg.StorageDriver)
	assert.Equal(t, "cicekci", cfg.R2Bucket)
}

func TestLoad_RejectsUnknownStorageDriver(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("JWT_SECRET", "s3cret-key-that-is-long-enough-ok")
	t.Setenv("WHATSAPP_NUMBER", "905551234567")
	t.Setenv("SITE_URL", "https://example.com")
	t.Setenv("STORAGE_DRIVER", "gcs")

	_, err := Load()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "STORAGE_DRIVER")
}
```

- [ ] **Step 5: Testi çalıştır**

Run: `go test ./pkg/config/ -v`
Expected: PASS — 8 test

- [ ] **Step 6: .env.example'ı güncelle**

```
DATABASE_URL=postgres://cicekci:cicekci@localhost:5433/cicekci?sslmode=disable
TEST_DATABASE_URL=postgres://cicekci:cicekci@localhost:5434/cicekci_test?sslmode=disable
JWT_SECRET=change-me-in-production-min-32-chars
PORT=8080
WHATSAPP_NUMBER=905551234567
SITE_URL=http://localhost:3000

# Görsel saklama: local (geliştirme) veya r2 (production)
STORAGE_DRIVER=local
UPLOAD_DIR=./uploads
UPLOAD_BASE_URL=http://localhost:8080/uploads

# STORAGE_DRIVER=r2 ise zorunlu
R2_ACCOUNT_ID=
R2_ACCESS_KEY_ID=
R2_SECRET_ACCESS_KEY=
R2_BUCKET=
R2_PUBLIC_URL=
```

- [ ] **Step 7: .gitignore'a uploads ekle**

```
/uploads
```

- [ ] **Step 8: Commit**

```bash
git add internal/image/r2.go pkg/config/ .env.example .gitignore go.mod go.sum
git commit -m "feat: R2 saklama implementasyonu ve storage config"
```

---

## Task 6: Admin görsel uçları

**Files:**
- Create: `internal/api/idare/image_view.go`, `internal/api/idare/image_handler.go`
- Modify: `internal/api/idare/router.go` — görsel rotaları + Deps'e ImgSvc
- Modify: `internal/api/idare/product_view.go` — ProductView'e Images alanı
- Modify: `internal/api/idare/product_handler.go` — delete akışında görsel temizliği
- Test: `internal/api/idare/image_handler_test.go`

**Interfaces:**
- Consumes: `image.Service`
- Produces:
  - `idare.ImageView{ID int64, URL400 string, URL1200 string, SortOrder int}`
  - `idare.Deps` alanı: `ImgSvc *image.Service`

- [ ] **Step 1: image_view.go yaz**

`internal/api/idare/image_view.go`:
```go
package idare

import "github.com/omerkoc/cicekci/internal/image"

// ImageView admin görsel gösterimi.
// image_key JSON'a çıkmaz — URL'ler ImageStore'dan üretilir (spec §4.1).
type ImageView struct {
	ID        int64  `json:"id"`
	URL400    string `json:"url_400"`
	URL1200   string `json:"url_1200"`
	SortOrder int    `json:"sort_order"`
}

func toImageView(svc *image.Service, img image.ProductImage) ImageView {
	return ImageView{
		ID:        img.ID,
		URL400:    svc.URL(img.ImageKey, image.Size400),
		URL1200:   svc.URL(img.ImageKey, image.Size1200),
		SortOrder: img.SortOrder,
	}
}

func toImageViews(svc *image.Service, list []image.ProductImage) []ImageView {
	out := make([]ImageView, 0, len(list))
	for _, img := range list {
		out = append(out, toImageView(svc, img))
	}
	return out
}
```

- [ ] **Step 2: image_handler.go yaz**

`internal/api/idare/image_handler.go`:
```go
package idare

import (
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
)

const maxUploadBytes = 10 * 1024 * 1024 // 10MB

type imageHandler struct {
	svc     *image.Service
	prodSvc *product.Service
}

// list GET /api/admin/products/:id/images
func (h *imageHandler) list(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	list, err := h.svc.ListByProduct(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toImageViews(h.svc, list))
}

// upload POST /api/admin/products/:id/images
// multipart/form-data, alan adı: "image"
func (h *imageHandler) upload(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	// Ürün var mı — yoksa yetim görsel yüklemeyelim.
	if _, err := h.prodSvc.GetByID(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}

	fileHeader, err := c.FormFile("image")
	if err != nil {
		return badRequest(c, "Görsel dosyası bulunamadı (alan adı: image)")
	}

	if fileHeader.Size > maxUploadBytes {
		return badRequest(c, fmt.Sprintf("Dosya çok büyük (en fazla %d MB)",
			maxUploadBytes/1024/1024))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return api.WriteError(c, fmt.Errorf("dosya aç: %w", err))
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes))
	if err != nil {
		return api.WriteError(c, fmt.Errorf("dosya oku: %w", err))
	}

	img, err := h.svc.Upload(c.Context(), int64(id), data)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toImageView(h.svc, *img))
}

// delete DELETE /api/admin/images/:id
func (h *imageHandler) delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	if err := h.svc.Delete(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type reorderRequest struct {
	ImageIDs []int64 `json:"image_ids"`
}

// reorder PATCH /api/admin/products/:id/images/order
// Body: {"image_ids": [3, 1, 2]} — ürünün TÜM görsellerini içermeli.
// İlk id yeni kapak olur.
func (h *imageHandler) reorder(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req reorderRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	if err := h.svc.Reorder(c.Context(), int64(id), req.ImageIDs); err != nil {
		return api.WriteError(c, err)
	}

	list, err := h.svc.ListByProduct(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toImageViews(h.svc, list))
}

func badRequest(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(api.ErrorResponse{
		Error: api.ErrorBody{Code: "invalid_input", Message: msg},
	})
}
```

- [ ] **Step 3: product_view.go'yu güncelle**

`internal/api/idare/product_view.go` — `ProductView`'e `Images` ekle ve dönüştürücüleri güncelle:
```go
package idare

import (
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
)

// ProductView admin ürün gösterimi — is_active DAHİL.
type ProductView struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	Description string      `json:"description"`
	Price       string      `json:"price"`
	IsActive    bool        `json:"is_active"`
	CategoryIDs []int64     `json:"category_ids"`
	Images      []ImageView `json:"images"`
}

func toProductView(p product.Product, imgSvc *image.Service, imgs []image.ProductImage) ProductView {
	return ProductView{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Price:       p.Price.StringFixed(2),
		IsActive:    p.IsActive,
		CategoryIDs: p.CategoryIDs,
		Images:      toImageViews(imgSvc, imgs),
	}
}

// toProductViews N+1 sorgu yapmaz — görseller tek sorguda gelir.
func toProductViews(list []product.Product, imgSvc *image.Service,
	grouped map[int64][]image.ProductImage) []ProductView {
	out := make([]ProductView, 0, len(list))
	for _, p := range list {
		out = append(out, toProductView(p, imgSvc, grouped[p.ID]))
	}
	return out
}
```

- [ ] **Step 4: product_handler.go'yu güncelle**

`internal/api/idare/product_handler.go` — struct'a `imgSvc` ekle:
```go
type productHandler struct {
	svc    *product.Service
	imgSvc *image.Service
}
```

`list` metodunu değiştir:
```go
func (h *productHandler) list(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 24)
	offset := 0
	if page := c.QueryInt("page", 1); page > 1 {
		offset = (page - 1) * limit
	}

	list, err := h.svc.ListAdmin(c.Context(), limit, offset)
	if err != nil {
		return api.WriteError(c, err)
	}

	ids := make([]int64, 0, len(list))
	for _, p := range list {
		ids = append(ids, p.ID)
	}
	grouped, err := h.imgSvc.ListByProducts(c.Context(), ids)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(toProductViews(list, h.imgSvc, grouped))
}
```

`get` metodunu değiştir:
```go
func (h *productHandler) get(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	p, err := h.svc.GetByID(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}

	imgs, err := h.imgSvc.ListByProduct(c.Context(), p.ID)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(toProductView(*p, h.imgSvc, imgs))
}
```

`create` ve `update` metotlarının son satırlarını değiştir — yeni ürünün görseli yoktur, güncellenmiş ürünün olabilir:
```go
// create sonunda:
	return c.Status(fiber.StatusCreated).JSON(
		toProductView(*p, h.imgSvc, nil))

// update sonunda:
	imgs, err := h.imgSvc.ListByProduct(c.Context(), p.ID)
	if err != nil {
		return api.WriteError(c, err)
	}
	return c.JSON(toProductView(*p, h.imgSvc, imgs))
```

`delete` metodunu değiştir — **görseller ürün silinmeden ÖNCE temizlenmeli** (spec §4.4):
```go
func (h *productHandler) delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	// Önce saklamadaki dosyaları temizle — ürün silinince product_images
	// CASCADE ile gider ve key'ler öğrenilemez hale gelir (spec §4.4).
	if err := h.imgSvc.DeleteAllForProduct(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}

	if err := h.svc.Delete(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```

Ayrıca dosyanın başındaki `import` bloğuna `"github.com/omerkoc/cicekci/internal/image"` eklenmeli, ve önceki `badRequest` tekrarları (inline `c.Status(...).JSON(...)` blokları) `badRequest(c, "Geçersiz id")` çağrılarıyla değiştirilmeli — helper artık `image_handler.go`'da tanımlı.

- [ ] **Step 5: router.go'yu güncelle**

`internal/api/idare/router.go`:
```go
package idare

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
)

type Deps struct {
	AuthSvc      *auth.Service
	CatSvc       *category.Service
	ProdSvc      *product.Service
	ImgSvc       *image.Service
	JWTSecret    string
	SecureCookie bool
}

// Register admin rotalarını bağlar. /login hariç hepsi JWT korumalı.
func Register(router fiber.Router, d Deps) {
	ah := &authHandler{svc: d.AuthSvc, secureCookie: d.SecureCookie}
	ch := &categoryHandler{svc: d.CatSvc}
	ph := &productHandler{svc: d.ProdSvc, imgSvc: d.ImgSvc}
	ih := &imageHandler{svc: d.ImgSvc, prodSvc: d.ProdSvc}

	router.Post("/login", ah.login)

	protected := router.Group("", auth.Middleware(d.JWTSecret))

	protected.Post("/logout", ah.logout)
	protected.Get("/me", ah.me)

	protected.Get("/products", ph.list)
	protected.Post("/products", ph.create)
	protected.Get("/products/:id", ph.get)
	protected.Patch("/products/:id", ph.update)
	protected.Delete("/products/:id", ph.delete)

	protected.Get("/products/:id/images", ih.list)
	protected.Post("/products/:id/images", ih.upload)
	protected.Patch("/products/:id/images/order", ih.reorder)
	protected.Delete("/images/:id", ih.delete)

	protected.Get("/categories", ch.list)
	protected.Post("/categories", ch.create)
	protected.Patch("/categories/:id", ch.update)
	protected.Get("/categories/:id/product-count", ch.productCount)
	protected.Delete("/categories/:id", ch.delete)
}
```

- [ ] **Step 6: Mevcut admin testini güncelle**

`internal/api/idare/product_handler_test.go` — `newTestAdminAPI` fonksiyonunda `Deps`'e `ImgSvc` eklenmeli:
```go
func newTestAdminAPI(t *testing.T) (*fiber.App, string) {
	t.Helper()
	pool := database.NewTestDB(t)

	authSvc := auth.NewService(auth.NewStore(pool), testSecret)
	require.NoError(t, authSvc.CreateAdmin(context.Background(), "cicekci", "test-sifre-123"))

	imgStore, err := image.NewLocalStore(t.TempDir(), "http://localhost:8080/uploads")
	require.NoError(t, err)

	app := fiber.New()
	Register(app.Group("/api/admin"), Deps{
		AuthSvc:      authSvc,
		CatSvc:       category.NewService(category.NewStore(pool)),
		ProdSvc:      product.NewService(product.NewStore(pool)),
		ImgSvc:       image.NewService(imgStore, image.NewDB(pool)),
		JWTSecret:    testSecret,
		SecureCookie: false,
	})

	token, err := auth.GenerateToken(1, "cicekci", testSecret)
	require.NoError(t, err)
	return app, token
}
```

`"github.com/omerkoc/cicekci/internal/image"` importunu eklemeyi unutma.

- [ ] **Step 7: Görsel handler testini yaz**

`internal/api/idare/image_handler_test.go`:
```go
package idare

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: 150, B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

// uploadRequest multipart görsel yükleme isteği kurar.
func uploadRequest(t *testing.T, url, token string, data []byte, fieldName, fileName string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile(fieldName, fileName)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, url, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "cicekci_token", Value: token})
	return req
}

// createProduct test için ürün oluşturur ve id döner.
func createProduct(t *testing.T, app *fiber.App, token, name string) int64 {
	t.Helper()
	resp, err := app.Test(authedRequest(http.MethodPost, "/api/admin/products",
		`{"name":"`+name+`","price":"500.00"}`, token))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	return view.ID
}

func TestImage_Upload(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")

	resp, err := app.Test(uploadRequest(t,
		"/api/admin/products/"+strconv.FormatInt(pid, 10)+"/images",
		token, makeTestJPEG(t, 1000, 800), "image", "foto.jpg"), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var view ImageView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	assert.Contains(t, view.URL400, "/400.webp")
	assert.Contains(t, view.URL1200, "/1200.webp")
	assert.Equal(t, 0, view.SortOrder)
}

func TestImage_Upload_RequiresAuth(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("image", "foto.jpg")
	require.NoError(t, err)
	_, err = part.Write(makeTestJPEG(t, 100, 100))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost,
		"/api/admin/products/"+strconv.FormatInt(pid, 10)+"/images", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := app.Test(req, -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestImage_Upload_RejectsPDF(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")

	resp, err := app.Test(uploadRequest(t,
		"/api/admin/products/"+strconv.FormatInt(pid, 10)+"/images",
		token, []byte("%PDF-1.4 bu bir PDF"), "image", "foto.jpg"), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestImage_Upload_WrongFieldName(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")

	resp, err := app.Test(uploadRequest(t,
		"/api/admin/products/"+strconv.FormatInt(pid, 10)+"/images",
		token, makeTestJPEG(t, 100, 100), "dosya", "foto.jpg"), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestImage_Upload_ProductNotFound(t *testing.T) {
	app, token := newTestAdminAPI(t)

	resp, err := app.Test(uploadRequest(t, "/api/admin/products/9999/images",
		token, makeTestJPEG(t, 100, 100), "image", "foto.jpg"), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestImage_List(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")
	path := "/api/admin/products/" + strconv.FormatInt(pid, 10) + "/images"
	_, err := app.Test(uploadRequest(t, path, token, makeTestJPEG(t, 800, 600), "image", "1.jpg"), -1)
	require.NoError(t, err)
	_, err = app.Test(uploadRequest(t, path, token, makeTestJPEG(t, 800, 600), "image", "2.jpg"), -1)
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodGet, path, "", token), -1)

	require.NoError(t, err)
	var views []ImageView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	require.Len(t, views, 2)
	assert.Equal(t, 0, views[0].SortOrder)
	assert.Equal(t, 1, views[1].SortOrder)
}

func TestImage_Delete(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")
	uploadResp, err := app.Test(uploadRequest(t,
		"/api/admin/products/"+strconv.FormatInt(pid, 10)+"/images",
		token, makeTestJPEG(t, 800, 600), "image", "foto.jpg"), -1)
	require.NoError(t, err)
	var img ImageView
	require.NoError(t, json.NewDecoder(uploadResp.Body).Decode(&img))

	resp, err := app.Test(authedRequest(http.MethodDelete,
		"/api/admin/images/"+strconv.FormatInt(img.ID, 10), "", token), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestImage_Reorder(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")
	path := "/api/admin/products/" + strconv.FormatInt(pid, 10) + "/images"

	r1, err := app.Test(uploadRequest(t, path, token, makeTestJPEG(t, 800, 600), "image", "1.jpg"), -1)
	require.NoError(t, err)
	var first ImageView
	require.NoError(t, json.NewDecoder(r1.Body).Decode(&first))

	r2, err := app.Test(uploadRequest(t, path, token, makeTestJPEG(t, 800, 600), "image", "2.jpg"), -1)
	require.NoError(t, err)
	var second ImageView
	require.NoError(t, json.NewDecoder(r2.Body).Decode(&second))

	body := `{"image_ids":[` + strconv.FormatInt(second.ID, 10) + `,` +
		strconv.FormatInt(first.ID, 10) + `]}`
	resp, err := app.Test(authedRequest(http.MethodPatch, path+"/order", body, token), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var views []ImageView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&views))
	assert.Equal(t, second.ID, views[0].ID, "kapak değişmeli")
}

func TestImage_Reorder_IncompleteListRejected(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")
	path := "/api/admin/products/" + strconv.FormatInt(pid, 10) + "/images"

	r1, err := app.Test(uploadRequest(t, path, token, makeTestJPEG(t, 800, 600), "image", "1.jpg"), -1)
	require.NoError(t, err)
	var first ImageView
	require.NoError(t, json.NewDecoder(r1.Body).Decode(&first))
	_, err = app.Test(uploadRequest(t, path, token, makeTestJPEG(t, 800, 600), "image", "2.jpg"), -1)
	require.NoError(t, err)

	body := `{"image_ids":[` + strconv.FormatInt(first.ID, 10) + `]}`
	resp, err := app.Test(authedRequest(http.MethodPatch, path+"/order", body, token), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// Ürün silinince görselleri de gider — yetim dosya kalmaz (spec §4.4).
func TestProduct_Delete_RemovesImages(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Silinecek")
	_, err := app.Test(uploadRequest(t,
		"/api/admin/products/"+strconv.FormatInt(pid, 10)+"/images",
		token, makeTestJPEG(t, 800, 600), "image", "foto.jpg"), -1)
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodDelete,
		"/api/admin/products/"+strconv.FormatInt(pid, 10), "", token), -1)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

// Ürün detayında görseller görünmeli.
func TestProduct_Get_IncludesImages(t *testing.T) {
	app, token := newTestAdminAPI(t)
	pid := createProduct(t, app, token, "Buket")
	_, err := app.Test(uploadRequest(t,
		"/api/admin/products/"+strconv.FormatInt(pid, 10)+"/images",
		token, makeTestJPEG(t, 800, 600), "image", "foto.jpg"), -1)
	require.NoError(t, err)

	resp, err := app.Test(authedRequest(http.MethodGet,
		"/api/admin/products/"+strconv.FormatInt(pid, 10), "", token), -1)

	require.NoError(t, err)
	var view ProductView
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	require.Len(t, view.Images, 1)
	assert.Contains(t, view.Images[0].URL400, "/400.webp")
}
```

`app.Test(req, -1)` — `-1` timeout'u kapatır. Görsel işleme birkaç yüz ms sürüyor, Fiber'in varsayılan 1 saniyelik test timeout'u yetmeyebilir.

`"github.com/gofiber/fiber/v2"` importunu eklemeyi unutma (`createProduct` helper'ı `*fiber.App` alıyor).

- [ ] **Step 8: Testi çalıştır**

Run: `go test ./internal/api/idare/ -v`
Expected: PASS — mevcut 7 + yeni 11 test

- [ ] **Step 9: Commit**

```bash
git add internal/api/idare/
git commit -m "feat: admin görsel uçları — upload, sıralama, silme"
```

---

## Task 7: Public API'ye görseller

**Files:**
- Modify: `internal/api/app/product_view.go` — ProductView'e Images
- Modify: `internal/api/app/product_handler.go` — imgSvc bağlama
- Modify: `internal/api/app/router.go` — Register imzası
- Test: `internal/api/app/product_handler_test.go` — güncelle

**Interfaces:**
- Produces: `app.ImageView{URL400 string, URL1200 string}`, `app.Register(router, catSvc, prodSvc, imgSvc)`

- [ ] **Step 1: product_view.go'yu güncelle**

`internal/api/app/product_view.go`:
```go
package app

import (
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
)

// ImageView public görsel gösterimi.
// id, sort_order ve image_key yok — müşterinin işine yaramaz.
type ImageView struct {
	URL400  string `json:"url_400"`
	URL1200 string `json:"url_1200"`
}

// ProductView public ürün gösterimi.
// is_active alanı KASITLI olarak yok — public'e sızmaz (spec §4.6).
// Price string olarak gider: JSON float precision sorununu önler.
type ProductView struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	Description string      `json:"description"`
	Price       string      `json:"price"`
	CategoryIDs []int64     `json:"category_ids"`
	Images      []ImageView `json:"images"`
}

func toImageViews(imgSvc *image.Service, list []image.ProductImage) []ImageView {
	out := make([]ImageView, 0, len(list))
	for _, img := range list {
		out = append(out, ImageView{
			URL400:  imgSvc.URL(img.ImageKey, image.Size400),
			URL1200: imgSvc.URL(img.ImageKey, image.Size1200),
		})
	}
	return out
}

func toProductView(p product.Product, imgSvc *image.Service, imgs []image.ProductImage) ProductView {
	return ProductView{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Price:       p.Price.StringFixed(2),
		CategoryIDs: p.CategoryIDs,
		Images:      toImageViews(imgSvc, imgs),
	}
}

// toProductViews N+1 sorgu yapmaz — görseller tek sorguda gelir.
func toProductViews(list []product.Product, imgSvc *image.Service,
	grouped map[int64][]image.ProductImage) []ProductView {
	out := make([]ProductView, 0, len(list))
	for _, p := range list {
		out = append(out, toProductView(p, imgSvc, grouped[p.ID]))
	}
	return out
}
```

- [ ] **Step 2: product_handler.go'yu güncelle**

`internal/api/app/product_handler.go`:
```go
package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
)

type productHandler struct {
	svc    *product.Service
	imgSvc *image.Service
}

// list GET /api/products?amac=&tip=&page=
func (h *productHandler) list(c *fiber.Ctx) error {
	f := product.Filter{
		Limit:  c.QueryInt("limit", 24),
		Offset: 0,
	}

	if page := c.QueryInt("page", 1); page > 1 {
		f.Offset = (page - 1) * f.Limit
	}
	if amac := c.Query("amac"); amac != "" {
		f.OccasionSlug = &amac
	}
	if tip := c.Query("tip"); tip != "" {
		f.TypeSlug = &tip
	}

	list, err := h.svc.ListPublic(c.Context(), f)
	if err != nil {
		return api.WriteError(c, err)
	}

	ids := make([]int64, 0, len(list))
	for _, p := range list {
		ids = append(ids, p.ID)
	}
	grouped, err := h.imgSvc.ListByProducts(c.Context(), ids)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(toProductViews(list, h.imgSvc, grouped))
}

// getBySlug GET /api/products/:slug
// Slug eskiyse 301 ile güncel URL'e yönlendirir (spec §4.2).
func (h *productHandler) getBySlug(c *fiber.Ctx) error {
	p, redirectTo, err := h.svc.GetPublicBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		return api.WriteError(c, err)
	}

	if redirectTo != "" {
		return c.Redirect("/api/products/"+redirectTo, fiber.StatusMovedPermanently)
	}

	imgs, err := h.imgSvc.ListByProduct(c.Context(), p.ID)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(toProductView(*p, h.imgSvc, imgs))
}
```

- [ ] **Step 3: router.go'yu güncelle**

`internal/api/app/router.go`:
```go
package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/category"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
)

// Register public rotaları bağlar. Auth yok — herkes erişebilir.
func Register(router fiber.Router, catSvc *category.Service,
	prodSvc *product.Service, imgSvc *image.Service) {
	ch := &categoryHandler{svc: catSvc}
	ph := &productHandler{svc: prodSvc, imgSvc: imgSvc}

	router.Get("/products", ph.list)
	router.Get("/products/:slug", ph.getBySlug)

	// /categories/featured, /categories/:slug'dan ÖNCE tanımlanmalı —
	// yoksa "featured" slug olarak yakalanır.
	router.Get("/categories/featured", ch.listFeatured)
	router.Get("/categories", ch.list)
	router.Get("/categories/:slug", ch.getBySlug)
}
```

- [ ] **Step 4: Public testini güncelle**

`internal/api/app/product_handler_test.go` — `newTestAPI` fonksiyonunu değiştir:
```go
func newTestAPI(t *testing.T) (*fiber.App, *product.Service, *category.Service) {
	t.Helper()
	pool := database.NewTestDB(t)
	prodSvc := product.NewService(product.NewStore(pool))
	catSvc := category.NewService(category.NewStore(pool))

	imgStore, err := image.NewLocalStore(t.TempDir(), "http://localhost:8080/uploads")
	require.NoError(t, err)
	imgSvc := image.NewService(imgStore, image.NewDB(pool))

	app := fiber.New()
	Register(app.Group("/api"), catSvc, prodSvc, imgSvc)
	return app, prodSvc, catSvc
}
```

`"github.com/omerkoc/cicekci/internal/image"` importunu ekle.

- [ ] **Step 5: Public görsel testi ekle**

`internal/api/app/product_handler_test.go` dosyasının sonuna:
```go
// Public görsel gösteriminde iç detay sızmamalı — image_key, id, sort_order yok.
func TestProductHandler_ImageViewHasNoInternalFields(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	ctx := context.Background()
	_, err := svc.Create(ctx, product.CreateInput{
		Name: "Buket", Price: mustPrice(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products/buket", nil))
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.NotContains(t, string(body), "image_key")
	assert.NotContains(t, string(body), "sort_order")
}

// Görseli olmayan ürün boş dizi döner, null değil — frontend'de
// v-for patlamasın.
func TestProductHandler_NoImagesReturnsEmptyArray(t *testing.T) {
	app, svc, _ := newTestAPI(t)
	_, err := svc.Create(context.Background(), product.CreateInput{
		Name: "Görselsiz", Price: mustPrice(t, "500"), IsActive: true,
	})
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/products/gorselsiz", nil))
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), `"images":[]`)
	assert.NotContains(t, string(body), `"images":null`)
}
```

- [ ] **Step 6: Testi çalıştır**

Run: `go test ./internal/api/app/ -v`
Expected: PASS — 9 test

- [ ] **Step 7: Commit**

```bash
git add internal/api/app/
git commit -m "feat: public API'ye görsel URL'leri"
```

---

## Task 8: Server main — ImageStore bağlama

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: main.go'yu güncelle**

`cmd/server/main.go` — servis oluşturma bloğuna ekle (`prodSvc` satırından sonra):
```go
	imgStore, err := newImageStore(cfg)
	if err != nil {
		log.Fatalf("görsel saklama: %v", err)
	}
	imgSvc := image.NewService(imgStore, image.NewDB(pool))
```

Route bağlama bloğunu değiştir:
```go
	apiGroup := f.Group("/api")
	app.Register(apiGroup, catSvc, prodSvc, imgSvc)
	idare.Register(apiGroup.Group("/admin"), idare.Deps{
		AuthSvc:      authSvc,
		CatSvc:       catSvc,
		ProdSvc:      prodSvc,
		ImgSvc:       imgSvc,
		JWTSecret:    cfg.JWTSecret,
		SecureCookie: isProduction,
	})

	// Local saklama modunda görselleri statik servis et.
	// R2 modunda bu gerekmez — CDN servis eder.
	if cfg.StorageDriver == "local" {
		f.Static("/uploads", cfg.UploadDir, fiber.Static{
			MaxAge: 31536000,
		})
	}
```

Dosyanın sonuna helper ekle:
```go
// newImageStore config'e göre saklama implementasyonunu seçer.
// Uygulama kodunun geri kalanı hangisi olduğunu bilmez (spec §4.4).
func newImageStore(cfg *config.Config) (image.Store, error) {
	switch cfg.StorageDriver {
	case "r2":
		return image.NewR2Store(image.R2Config{
			AccountID:       cfg.R2AccountID,
			AccessKeyID:     cfg.R2AccessKey,
			SecretAccessKey: cfg.R2SecretKey,
			Bucket:          cfg.R2Bucket,
			PublicURL:       cfg.R2PublicURL,
		})
	case "local":
		return image.NewLocalStore(cfg.UploadDir, cfg.UploadBaseURL)
	default:
		return nil, fmt.Errorf("bilinmeyen STORAGE_DRIVER: %q", cfg.StorageDriver)
	}
}
```

`"fmt"` ve `"github.com/omerkoc/cicekci/internal/image"` importlarını ekle.

- [ ] **Step 2: Sunucuyu başlat**

```bash
export DATABASE_URL="postgres://cicekci:cicekci@localhost:5433/cicekci?sslmode=disable"
export JWT_SECRET="local-development-secret-32-chars!"
export WHATSAPP_NUMBER="905551234567"
export SITE_URL="http://localhost:3000"
export STORAGE_DRIVER=local
make run
```
Expected: sunucu ayağa kalkar, hata yok

- [ ] **Step 3: Login ve ürün oluştur**

```bash
curl -s -c /tmp/c.txt -X POST localhost:8080/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"cicekci","password":"test-sifre-123"}'

curl -s -b /tmp/c.txt -X POST localhost:8080/api/admin/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Buket","price":"500.00"}'
```
Expected: `201`, `"images":[]`

- [ ] **Step 4: Gerçek bir fotoğraf yükle**

```bash
# Test için bir JPEG indir (veya kendi fotoğrafını kullan)
curl -s -o /tmp/test.jpg "https://picsum.photos/2000/1500"
ls -lh /tmp/test.jpg

curl -s -b /tmp/c.txt -X POST localhost:8080/api/admin/products/1/images \
  -F "image=@/tmp/test.jpg"
```
Expected: `201`, `url_400` ve `url_1200` dolu

- [ ] **Step 5: Üretilen dosyaları doğrula**

```bash
find ./uploads -type f -exec ls -lh {} \;
file ./uploads/products/*/400.webp
```
Expected: iki dosya (400.webp, 1200.webp), `file` çıktısı `RIFF ... Web/P image`
400.webp orijinal 2000px JPEG'den belirgin küçük olmalı.

- [ ] **Step 6: Görseli tarayıcıda gör**

```bash
curl -s -I localhost:8080/uploads/products/$(ls ./uploads/products | head -1)/400.webp
```
Expected: `HTTP/1.1 200 OK`, `Content-Type: image/webp`

- [ ] **Step 7: Public uçta görsel görünüyor mu**

```bash
curl -s localhost:8080/api/products/test-buket | head -c 400
```
Expected: `"images":[{"url_400":"http://localhost:8080/uploads/products/.../400.webp",...}]`

- [ ] **Step 8: Geçersiz dosya reddediliyor mu**

```bash
echo "bu bir metin dosyası, görsel değil" > /tmp/sahte.jpg
curl -s -b /tmp/c.txt -X POST localhost:8080/api/admin/products/1/images \
  -F "image=@/tmp/sahte.jpg"
```
Expected: `400`, `"Sadece JPEG, PNG veya WebP"` içeren mesaj — uzantı `.jpg` olmasına rağmen reddedilmeli

- [ ] **Step 9: Ürün silinince dosyalar gidiyor mu**

```bash
curl -s -b /tmp/c.txt -X DELETE localhost:8080/api/admin/products/1
find ./uploads -type f | wc -l
```
Expected: `0` — dosyalar temizlenmiş

- [ ] **Step 10: Tüm testleri çalıştır**

```bash
export TEST_DATABASE_URL="postgres://cicekci:cicekci@localhost:5434/cicekci_test?sslmode=disable"
make test
```
Expected: tüm paketler `ok`

- [ ] **Step 11: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: ImageStore'u sunucuya bağla, local statik servis"
```

---

## Plan 2 Bitiş Kriterleri

- [ ] `make test` tüm paketlerde geçiyor
- [ ] JPEG, PNG ve WebP yüklenebiliyor; PDF/GIF reddediliyor (içerikten tespit)
- [ ] Yüklenen her görsel 400px ve 1200px WebP olarak üretiliyor
- [ ] 2000px'lik JPEG → 400px WebP belirgin küçük
- [ ] Küçük görsel büyütülmüyor
- [ ] Saklama hatası olursa DB'ye kayıt atılmıyor, yazılan dosyalar temizleniyor
- [ ] Görsel silince dosyalar da gidiyor
- [ ] Ürün silince tüm görselleri gidiyor
- [ ] Sıralama değiştirince kapak değişiyor
- [ ] Public API'de `image_key` görünmüyor, sadece URL'ler
- [ ] `STORAGE_DRIVER=local` ve `r2` arasında geçiş sadece config değişikliği

**Sonraki:** Plan 3 — Nuxt public site. Bu plan, Plan 1+2 uygulanıp API'nin gerçek davranışı görüldükten sonra yazılacak.
