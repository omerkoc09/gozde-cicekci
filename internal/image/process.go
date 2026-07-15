package image

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"

	"github.com/disintegration/imaging"

	// Decoder kayıtları — image.Decode bunlar olmadan formatı tanımaz.
	_ "image/jpeg"
	_ "image/png"
)

// ErrUnsupportedFormat desteklenmeyen girdi formatı.
var ErrUnsupportedFormat = errors.New("desteklenmeyen görsel formatı")

// jpegQuality 1-100. 82 gözle fark edilmeyen kayıpla iyi sıkıştırma verir.
const jpegQuality = 82

// DetectFormat görsel formatını İÇERİKTEN tespit eder, uzantıdan değil.
// Uzantı yalan söyleyebilir: .jpg uzantılı bir dosya PDF olabilir.
// Kabul edilen girdiler: jpeg, png.
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
	default:
		return "", fmt.Errorf("%w: %s (sadece JPEG veya PNG)",
			ErrUnsupportedFormat, contentType)
	}
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

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("görsel decode: %w", err)
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
