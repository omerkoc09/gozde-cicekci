package image

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"

	"github.com/disintegration/imaging"
	"github.com/omerkoc/cicekci/pkg/errorsx"

	// Decoder kayıtları — image.Decode bunlar olmadan formatı tanımaz.
	// webp yalnızca DECODE eder; çıktı her zaman JPEG (bkz. Process).
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

// ErrUnsupportedFormat desteklenmeyen girdi formatı.
var ErrUnsupportedFormat = errors.New("desteklenmeyen görsel formatı")

// jpegQuality 1-100. 82 gözle fark edilmeyen kayıpla iyi sıkıştırma verir.
const jpegQuality = 82

// DetectFormat görsel formatını İÇERİKTEN tespit eder, uzantıdan değil.
// Uzantı yalan söyleyebilir: .jpg uzantılı bir dosya PDF olabilir.
// Kabul edilen girdiler: jpeg, png, webp.
//
// webp yalnızca GİRDİ olarak kabul edilir — Process çıktıyı her zaman JPEG
// verir. Yani panele webp yüklenebilir ama site .jpg servis etmeye devam eder.
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

// CheckMinWidth görselin en az minWidth piksel genişlikte olduğunu doğrular.
// Sistem küçük görselleri BÜYÜTMEZ (bulanıklaşır), o yüzden hedef boyutlardan
// (ör. ürün için 1200px) küçük bir foto yüklenirse kart/detay bulanık kalır.
// Bunu yükleme anında reddedip esnafa anlaşılır bir mesaj döndürüyoruz.
//
// Decode hatası da ErrInvalidInput ile sarılır: bozuk/eksik dosya sunucu
// hatası değil, kullanıcının seçtiği dosyanın sorunudur (API 400 dönsün).
func CheckMinWidth(data []byte, minWidth int) error {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%w: görsel okunamadı (dosya bozuk veya desteklenmeyen)", errorsx.ErrInvalidInput)
	}

	if cfg.Width < minWidth {
		return fmt.Errorf("%w: görsel çok küçük (%dpx) — en az %dpx genişliğinde, "+
			"kaliteli bir fotoğraf yükleyin", errorsx.ErrInvalidInput, cfg.Width, minWidth)
	}

	return nil
}

// Process görseli decode eder, hedef genişliğe küçültür ve JPEG olarak encode eder.
// Küçük görseller büyütülmez — bulanıklaşır ve dosya boyutu boşuna büyür.
// En-boy oranı korunur.
//
// Asıl kazanç formatta değil boyutta: esnafın attığı 4000px'lik fotoğraf
// liste kartında 400px'e düşünce veri ~100 kat azalıyor.
func Process(data []byte, size Size) ([]byte, error) {
	if !size.Valid() {
		return nil, fmt.Errorf("geçersiz boyut: %q", size)
	}

	if _, err := DetectFormat(data); err != nil {
		return nil, err
	}

	// Başlık doğru ama gövde bozuksa decode patlar (yarım inen dosya, kesilmiş
	// upload). Bu sunucu hatası DEĞİL, kullanıcının seçtiği dosyanın sorunu —
	// ErrUnsupportedFormat ile sarılıyor ki API 500 değil 400 dönsün.
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: dosya bozuk veya eksik (%v)", ErrUnsupportedFormat, err)
	}

	resized := resizeToWidth(src, size.Width())

	// PNG'nin şeffaf alanları JPEG'de siyah çıkar — beyaz zemine harmanla.
	// Paste değil Overlay: Paste alfayı yok sayıp kopyalar, Overlay harmanlar.
	flattened := imaging.New(
		resized.Bounds().Dx(), resized.Bounds().Dy(),
		image.White.C,
	)
	flattened = imaging.Overlay(flattened, resized, image.Pt(0, 0), 1.0)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, flattened, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("jpeg encode: %w", err)
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
