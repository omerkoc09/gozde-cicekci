package slider

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	imagesvc "github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestService gerçek LocalStore ile kurar — dosyaların gerçekten yazılıp
// silindiğini diskten doğrulayabilmek için. Yetim dosya hataları ancak
// böyle yakalanır.
func newTestService(t *testing.T) (*Service, string, context.Context) {
	t.Helper()
	pool := database.NewTestDB(t)
	dir := t.TempDir()

	store, err := imagesvc.NewLocalStore(dir, "http://localhost:8080/uploads")
	require.NoError(t, err)

	imgSvc := imagesvc.NewService(store, imagesvc.NewDB(pool))
	return NewService(NewStore(pool), imgSvc), dir, context.Background()
}

func makeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

// slideDirExists slider görselinin diskte durup durmadığını söyler.
func slideDirExists(t *testing.T, dir, key string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(dir, "slider", key))
	return err == nil
}

func validInput() CreateInput {
	return CreateInput{Title: "Bahar Koleksiyonu", Subtitle: "Yeni sezon", IsActive: true}
}

func TestCreate_StoresAllSliderSizes(t *testing.T) {
	svc, dir, ctx := newTestService(t)

	slide, err := svc.Create(ctx, validInput(), makeJPEG(t, 2400, 1200))

	require.NoError(t, err)
	assert.Equal(t, "Bahar Koleksiyonu", slide.Title)
	assert.NotEmpty(t, slide.ImageKey)

	// Slider 1920'yi de üretmeli — ürün görsellerinden farkı bu.
	for _, size := range imagesvc.SliderSizes {
		path := filepath.Join(dir, "slider", slide.ImageKey, string(size)+".jpg")
		_, err := os.Stat(path)
		assert.NoError(t, err, "%s boyutu yazılmalı", size)
	}
}

// Slider dosyaları ürünlerinkiyle karışmamalı — ayrı prefix.
func TestCreate_WritesUnderSliderPrefix(t *testing.T) {
	svc, dir, ctx := newTestService(t)

	slide, err := svc.Create(ctx, validInput(), makeJPEG(t, 2400, 1200))
	require.NoError(t, err)

	assert.True(t, slideDirExists(t, dir, slide.ImageKey))
	_, err = os.Stat(filepath.Join(dir, "products", slide.ImageKey))
	assert.True(t, os.IsNotExist(err), "slider görseli products/ altına yazılmamalı")
}

func TestCreate_RejectsEmptyTitle(t *testing.T) {
	svc, _, ctx := newTestService(t)

	in := validInput()
	in.Title = "   "

	_, err := svc.Create(ctx, in, makeJPEG(t, 2400, 1200))

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

// Görselsiz slayt gösterilemez — oluşturulmasına da izin verilmemeli.
func TestCreate_RejectsMissingImage(t *testing.T) {
	svc, _, ctx := newTestService(t)

	_, err := svc.Create(ctx, validInput(), nil)

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

// Panelden webp yüklenebilmeli — esnafın elindeki görsel webp olabilir.
// Fixture image paketinde: webp encoder olmadığı için üretilemiyor, okunuyor.
// Slider görselleri tam ekran arka plan — en büyük boyut 1920px üretiliyor,
// o yüzden yükleme de en az 1920px genişlik ister. Küçük görsel büyütülünce
// bulanık kalırdı. (webp→jpeg format dönüşümü image paketinde ayrıca test
// ediliyor; fixture 600px olduğu için burada boyut kontrolüne takılır.)
func TestCreate_RejectsTooSmall(t *testing.T) {
	svc, _, ctx := newTestService(t)
	data, err := os.ReadFile("../image/testdata/sample.webp") // 600x400
	require.NoError(t, err)

	_, err = svc.Create(ctx, validInput(), data)

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestCreate_RejectsNonImageData(t *testing.T) {
	svc, _, ctx := newTestService(t)

	_, err := svc.Create(ctx, validInput(), []byte("bu bir PDF değil, hiç değil"))

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestCreate_TrimsWhitespace(t *testing.T) {
	svc, _, ctx := newTestService(t)

	slide, err := svc.Create(ctx, CreateInput{
		Title:    "  Bahar  ",
		Subtitle: "  Yeni sezon  ",
	}, makeJPEG(t, 2400, 1200))

	require.NoError(t, err)
	assert.Equal(t, "Bahar", slide.Title)
	assert.Equal(t, "Yeni sezon", slide.Subtitle)
}

func TestNextSortOrder_StartsAtZeroThenAppends(t *testing.T) {
	svc, _, ctx := newTestService(t)

	first, err := svc.NextSortOrder(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, first, "hiç slayt yokken 0'dan başlamalı")

	in := validInput()
	in.SortOrder = 5
	_, err = svc.Create(ctx, in, makeJPEG(t, 2400, 1200))
	require.NoError(t, err)

	next, err := svc.NextSortOrder(ctx)
	require.NoError(t, err)
	assert.Equal(t, 6, next, "mevcut en büyük sıranın bir fazlası olmalı")
}

func TestListPublic_OnlyActiveInSortOrder(t *testing.T) {
	svc, _, ctx := newTestService(t)

	mk := func(title string, active bool, order int) {
		_, err := svc.Create(ctx, CreateInput{
			Title: title, IsActive: active, SortOrder: order,
		}, makeJPEG(t, 2400, 1200))
		require.NoError(t, err)
	}
	mk("Ikinci", true, 1)
	mk("Pasif", false, 0)
	mk("Birinci", true, 0)

	list, err := svc.ListPublic(ctx)

	require.NoError(t, err)
	require.Len(t, list, 2, "pasif slayt gelmemeli")
	assert.Equal(t, "Birinci", list[0].Title)
	assert.Equal(t, "Ikinci", list[1].Title, "sort_order'a göre sıralanmalı")
}

func TestListAdmin_IncludesInactive(t *testing.T) {
	svc, _, ctx := newTestService(t)

	in := validInput()
	in.IsActive = false
	_, err := svc.Create(ctx, in, makeJPEG(t, 2400, 1200))
	require.NoError(t, err)

	list, err := svc.ListAdmin(ctx)

	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestUpdate_PartialLeavesOtherFields(t *testing.T) {
	svc, _, ctx := newTestService(t)
	slide, err := svc.Create(ctx, validInput(), makeJPEG(t, 2400, 1200))
	require.NoError(t, err)

	newTitle := "Yaz Koleksiyonu"
	updated, err := svc.Update(ctx, slide.ID, UpdateInput{Title: &newTitle})

	require.NoError(t, err)
	assert.Equal(t, "Yaz Koleksiyonu", updated.Title)
	assert.Equal(t, slide.Subtitle, updated.Subtitle, "dokunulmayan alan korunmalı")
	assert.Equal(t, slide.ImageKey, updated.ImageKey, "metin güncellemesi görseli değiştirmemeli")
}

func TestUpdate_RejectsEmptyTitle(t *testing.T) {
	svc, _, ctx := newTestService(t)
	slide, err := svc.Create(ctx, validInput(), makeJPEG(t, 2400, 1200))
	require.NoError(t, err)

	empty := "  "
	_, err = svc.Update(ctx, slide.ID, UpdateInput{Title: &empty})

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestUpdate_MissingSlide(t *testing.T) {
	svc, _, ctx := newTestService(t)

	title := "yok"
	_, err := svc.Update(ctx, 9999, UpdateInput{Title: &title})

	assert.ErrorIs(t, err, errorsx.ErrNotFound)
}

// Görsel değişince eski dosya yetim kalmamalı.
func TestReplaceImage_DeletesOldFile(t *testing.T) {
	svc, dir, ctx := newTestService(t)
	slide, err := svc.Create(ctx, validInput(), makeJPEG(t, 2400, 1200))
	require.NoError(t, err)
	oldKey := slide.ImageKey

	updated, err := svc.ReplaceImage(ctx, slide.ID, makeJPEG(t, 2400, 1200))

	require.NoError(t, err)
	assert.NotEqual(t, oldKey, updated.ImageKey, "yeni key üretilmeli")
	assert.True(t, slideDirExists(t, dir, updated.ImageKey), "yeni görsel durmalı")
	assert.False(t, slideDirExists(t, dir, oldKey), "eski görsel silinmeli")
}

// Geçersiz görsel gelirse mevcut görsel korunmalı — slayt görselsiz kalmasın.
func TestReplaceImage_InvalidKeepsCurrent(t *testing.T) {
	svc, dir, ctx := newTestService(t)
	slide, err := svc.Create(ctx, validInput(), makeJPEG(t, 2400, 1200))
	require.NoError(t, err)

	_, err = svc.ReplaceImage(ctx, slide.ID, []byte("görsel değil"))

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
	after, err := svc.GetByID(ctx, slide.ID)
	require.NoError(t, err)
	assert.Equal(t, slide.ImageKey, after.ImageKey, "görsel değişmemeli")
	assert.True(t, slideDirExists(t, dir, slide.ImageKey), "eski görsel durmalı")
}

func TestReplaceImage_MissingSlide(t *testing.T) {
	svc, _, ctx := newTestService(t)

	_, err := svc.ReplaceImage(ctx, 9999, makeJPEG(t, 2400, 1200))

	assert.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestDelete_RemovesSlideAndFile(t *testing.T) {
	svc, dir, ctx := newTestService(t)
	slide, err := svc.Create(ctx, validInput(), makeJPEG(t, 2400, 1200))
	require.NoError(t, err)

	require.NoError(t, svc.Delete(ctx, slide.ID))

	_, err = svc.GetByID(ctx, slide.ID)
	assert.ErrorIs(t, err, errorsx.ErrNotFound)
	assert.False(t, slideDirExists(t, dir, slide.ImageKey), "görsel de silinmeli")
}

func TestDelete_MissingSlide(t *testing.T) {
	svc, _, ctx := newTestService(t)

	err := svc.Delete(ctx, 9999)

	assert.ErrorIs(t, err, errorsx.ErrNotFound)
}
