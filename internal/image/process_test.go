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

// Çıktı her zaman JPEG — girdi ne olursa olsun.
func TestProcess_JPEGInputStaysJPEG(t *testing.T) {
	out, err := Process(makeJPEG(t, 2000, 1500), Size400)

	require.NoError(t, err)
	require.NotEmpty(t, out)
	format, err := DetectFormat(out)
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)
}

func TestProcess_PNGInputBecomesJPEG(t *testing.T) {
	out, err := Process(makePNG(t, 800, 600), Size1200)

	require.NoError(t, err)
	format, err := DetectFormat(out)
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format, "PNG girdi de JPEG çıktı vermeli")
}

// Şeffaf PNG'nin alfa kanalı JPEG'de siyah çıkmamalı — beyaz zemine basılır.
func TestProcess_TransparentPNGGetsWhiteBackground(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	// Tamamen şeffaf bırak — hiç Set çağırmıyoruz, alfa 0 kalıyor.
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	out, err := Process(buf.Bytes(), Size400)

	require.NoError(t, err)
	decoded, _, err := image.Decode(bytes.NewReader(out))
	require.NoError(t, err)
	r, g, b, _ := decoded.At(50, 50).RGBA()
	assert.Greater(t, int(r>>8), 200, "şeffaf alan beyaz olmalı, siyah değil")
	assert.Greater(t, int(g>>8), 200)
	assert.Greater(t, int(b>>8), 200)
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

// Asıl kazanç: 2000px'lik fotoğraf 400px'e düşünce dosya çok küçülmeli.
// Format değil boyut kazandırıyor.
func TestProcess_OutputMuchSmallerThanOriginal(t *testing.T) {
	original := makeJPEG(t, 2000, 1500)

	out, err := Process(original, Size400)

	require.NoError(t, err)
	assert.Less(t, len(out), len(original)/2,
		"400px çıktı, 2000px orijinalin yarısından küçük olmalı")
}
