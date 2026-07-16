package category

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

func newTestService(t *testing.T) (*Service, context.Context) {
	svc, _, ctx := newTestServiceWithDir(t)
	return svc, ctx
}

// newTestServiceWithDir görsel testleri için: dosyaların gerçekten yazılıp
// silindiğini diskten doğrulayabilmek adına gerçek LocalStore kullanılıyor.
func newTestServiceWithDir(t *testing.T) (*Service, string, context.Context) {
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
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

func katGorselVar(t *testing.T, dir, key string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(dir, "categories", key))
	return err == nil
}

func TestService_Create(t *testing.T) {
	svc, ctx := newTestService(t)

	c, err := svc.Create(ctx, CreateInput{
		Name: "Doğum Günü", Axis: AxisOccasion, IsActive: true, IsFeatured: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "Doğum Günü", c.Name)
	assert.Equal(t, "dogum-gunu", c.Slug)
	assert.Equal(t, AxisOccasion, c.Axis)
	assert.True(t, c.IsFeatured)
}

func TestService_Create_InvalidAxis(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.Create(ctx, CreateInput{Name: "Test", Axis: "gecersiz"})

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_EmptyName(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.Create(ctx, CreateInput{Name: "  ", Axis: AxisType})

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_DuplicateSlugGetsSuffix(t *testing.T) {
	svc, ctx := newTestService(t)

	first, err := svc.Create(ctx, CreateInput{Name: "Buket", Axis: AxisType})
	require.NoError(t, err)
	second, err := svc.Create(ctx, CreateInput{Name: "Buket", Axis: AxisOccasion})
	require.NoError(t, err)

	assert.Equal(t, "buket", first.Slug)
	assert.Equal(t, "buket-2", second.Slug)
}

func TestService_ListPublic_HidesInactive(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{Name: "Doğum Günü", Axis: AxisOccasion, IsActive: true})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{Name: "Anneler Günü", Axis: AxisOccasion, IsActive: false})
	require.NoError(t, err)

	list, err := svc.ListPublic(ctx, nil)

	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Doğum Günü", list[0].Name)
}

func TestService_ListPublic_FiltersByAxis(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{Name: "Doğum Günü", Axis: AxisOccasion, IsActive: true})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{Name: "Buket", Axis: AxisType, IsActive: true})
	require.NoError(t, err)

	occasion := AxisOccasion
	list, err := svc.ListPublic(ctx, &occasion)

	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Doğum Günü", list[0].Name)
}

// Spec §4.1: is_active=false her şeyi ezer — pasif kategori featured olsa bile görünmez.
func TestService_ListFeatured_InactiveOverridesFeatured(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "Anneler Günü", Axis: AxisOccasion, IsActive: false, IsFeatured: true,
	})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{
		Name: "Doğum Günü", Axis: AxisOccasion, IsActive: true, IsFeatured: true,
	})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{
		Name: "Taziye", Axis: AxisOccasion, IsActive: true, IsFeatured: false,
	})
	require.NoError(t, err)

	list, err := svc.ListFeatured(ctx, nil)

	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Doğum Günü", list[0].Name)
}

// Ana sayfa "Özel Günler" ve "Çiçek Türlerine Göre" bölümlerini ayrı çekiyor;
// eksen filtresi olmazsa iki bölüm de aynı kategorileri gösterirdi.
func TestService_ListFeatured_FiltersByAxis(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "Sevgiliye", Axis: AxisOccasion, IsActive: true, IsFeatured: true,
	})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{
		Name: "Orkideler", Axis: AxisType, IsActive: true, IsFeatured: true,
	})
	require.NoError(t, err)

	occasion, err := svc.ListFeatured(ctx, ptr(AxisOccasion))
	require.NoError(t, err)
	require.Len(t, occasion, 1)
	assert.Equal(t, "Sevgiliye", occasion[0].Name)

	tip, err := svc.ListFeatured(ctx, ptr(AxisType))
	require.NoError(t, err)
	require.Len(t, tip, 1)
	assert.Equal(t, "Orkideler", tip[0].Name)

	// axis yoksa ikisi de gelir — eski davranış korunuyor.
	hepsi, err := svc.ListFeatured(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, hepsi, 2)
}

func TestService_ListFeatured_RejectsInvalidAxis(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.ListFeatured(ctx, ptr(Axis("bilinmeyen")))

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func ptr[T any](v T) *T { return &v }

// --- kart görseli ---

func TestService_ReplaceImage_StoresCategorySizes(t *testing.T) {
	svc, dir, ctx := newTestServiceWithDir(t)
	cat, err := svc.Create(ctx, CreateInput{Name: "Orkideler", Axis: AxisType, IsActive: true})
	require.NoError(t, err)

	updated, err := svc.ReplaceImage(ctx, cat.ID, makeJPEG(t, 1800, 2400))

	require.NoError(t, err)
	require.NotEmpty(t, updated.ImageKey)
	// Kategori kartı 400 ve 900 kullanıyor; 1920 üretmek boşuna bayt olurdu.
	for _, size := range imagesvc.CategorySizes {
		_, err := os.Stat(filepath.Join(dir, "categories", updated.ImageKey, string(size)+".jpg"))
		assert.NoError(t, err, "%s boyutu yazılmalı", size)
	}
}

// Kategori görselleri slider/ürün dosyalarıyla karışmamalı.
func TestService_ReplaceImage_UsesCategoryPrefix(t *testing.T) {
	svc, dir, ctx := newTestServiceWithDir(t)
	cat, err := svc.Create(ctx, CreateInput{Name: "Buketler", Axis: AxisType, IsActive: true})
	require.NoError(t, err)

	updated, err := svc.ReplaceImage(ctx, cat.ID, makeJPEG(t, 900, 1200))
	require.NoError(t, err)

	assert.True(t, katGorselVar(t, dir, updated.ImageKey))
	_, err = os.Stat(filepath.Join(dir, "slider", updated.ImageKey))
	assert.True(t, os.IsNotExist(err), "kategori görseli slider/ altına yazılmamalı")
}

// Yeni görsel yüklenince eskisi yetim kalmamalı.
func TestService_ReplaceImage_DeletesOldFile(t *testing.T) {
	svc, dir, ctx := newTestServiceWithDir(t)
	cat, err := svc.Create(ctx, CreateInput{Name: "Kutuda Çiçek", Axis: AxisType, IsActive: true})
	require.NoError(t, err)

	first, err := svc.ReplaceImage(ctx, cat.ID, makeJPEG(t, 900, 1200))
	require.NoError(t, err)
	oldKey := first.ImageKey

	second, err := svc.ReplaceImage(ctx, cat.ID, makeJPEG(t, 900, 1200))
	require.NoError(t, err)

	assert.NotEqual(t, oldKey, second.ImageKey)
	assert.True(t, katGorselVar(t, dir, second.ImageKey), "yeni görsel durmalı")
	assert.False(t, katGorselVar(t, dir, oldKey), "eski görsel silinmeli")
}

// Geçersiz dosya gelirse mevcut görsel korunmalı.
func TestService_ReplaceImage_InvalidKeepsCurrent(t *testing.T) {
	svc, dir, ctx := newTestServiceWithDir(t)
	cat, err := svc.Create(ctx, CreateInput{Name: "Tasarım", Axis: AxisType, IsActive: true})
	require.NoError(t, err)
	ok, err := svc.ReplaceImage(ctx, cat.ID, makeJPEG(t, 900, 1200))
	require.NoError(t, err)

	_, err = svc.ReplaceImage(ctx, cat.ID, []byte("görsel değil"))

	assert.ErrorIs(t, err, errorsx.ErrInvalidInput)
	after, err := svc.GetByID(ctx, cat.ID)
	require.NoError(t, err)
	assert.Equal(t, ok.ImageKey, after.ImageKey, "görsel değişmemeli")
	assert.True(t, katGorselVar(t, dir, ok.ImageKey))
}

func TestService_DeleteImage_RemovesFileAndKey(t *testing.T) {
	svc, dir, ctx := newTestServiceWithDir(t)
	cat, err := svc.Create(ctx, CreateInput{Name: "Orkide", Axis: AxisType, IsActive: true})
	require.NoError(t, err)
	withImg, err := svc.ReplaceImage(ctx, cat.ID, makeJPEG(t, 900, 1200))
	require.NoError(t, err)

	cleared, err := svc.DeleteImage(ctx, cat.ID)

	require.NoError(t, err)
	assert.Empty(t, cleared.ImageKey, "key temizlenmeli")
	assert.False(t, katGorselVar(t, dir, withImg.ImageKey), "dosya silinmeli")
}

// Görseli olmayan kategoride silme hata değil — idempotent.
func TestService_DeleteImage_NoImageIsNoError(t *testing.T) {
	svc, _, ctx := newTestServiceWithDir(t)
	cat, err := svc.Create(ctx, CreateInput{Name: "Görselsiz", Axis: AxisType, IsActive: true})
	require.NoError(t, err)

	_, err = svc.DeleteImage(ctx, cat.ID)

	assert.NoError(t, err)
}

// Kategori silinince görseli de gitmeli — yoksa dosya yetim kalır.
func TestService_Delete_RemovesImageFile(t *testing.T) {
	svc, dir, ctx := newTestServiceWithDir(t)
	cat, err := svc.Create(ctx, CreateInput{Name: "Silinecek", Axis: AxisType, IsActive: true})
	require.NoError(t, err)
	withImg, err := svc.ReplaceImage(ctx, cat.ID, makeJPEG(t, 900, 1200))
	require.NoError(t, err)

	require.NoError(t, svc.Delete(ctx, cat.ID))

	assert.False(t, katGorselVar(t, dir, withImg.ImageKey), "kategori görseli de silinmeli")
}

// Görseli olmayan kategoriler boş key döner — frontend yedek görsele düşer.
func TestService_Create_HasNoImageKey(t *testing.T) {
	svc, ctx := newTestService(t)

	cat, err := svc.Create(ctx, CreateInput{Name: "Yeni", Axis: AxisType, IsActive: true})

	require.NoError(t, err)
	assert.Empty(t, cat.ImageKey)
}

func TestService_ListAdmin_ShowsInactive(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{Name: "Aktif", Axis: AxisType, IsActive: true})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{Name: "Pasif", Axis: AxisType, IsActive: false})
	require.NoError(t, err)

	list, err := svc.ListAdmin(ctx)

	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestService_Update_PartialFields(t *testing.T) {
	svc, ctx := newTestService(t)
	c, err := svc.Create(ctx, CreateInput{
		Name: "Buket", Axis: AxisType, IsActive: true, IsFeatured: false,
	})
	require.NoError(t, err)

	featured := true
	updated, err := svc.Update(ctx, c.ID, UpdateInput{IsFeatured: &featured})

	require.NoError(t, err)
	assert.Equal(t, "Buket", updated.Name, "isim değişmemeli")
	assert.True(t, updated.IsFeatured)
	assert.True(t, updated.IsActive, "is_active değişmemeli")
}

// Spec §4.2: kategori slug'ı isim değişince güncellenmez — kategori URL'leri sabit kalır.
func TestService_Update_NameDoesNotChangeSlug(t *testing.T) {
	svc, ctx := newTestService(t)
	c, err := svc.Create(ctx, CreateInput{Name: "Buket", Axis: AxisType, IsActive: true})
	require.NoError(t, err)

	newName := "Gül Buketi"
	updated, err := svc.Update(ctx, c.ID, UpdateInput{Name: &newName})

	require.NoError(t, err)
	assert.Equal(t, "Gül Buketi", updated.Name)
	assert.Equal(t, "buket", updated.Slug, "slug sabit kalmalı")
}

func TestService_Update_NotFound(t *testing.T) {
	svc, ctx := newTestService(t)

	name := "Yok"
	_, err := svc.Update(ctx, 9999, UpdateInput{Name: &name})

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestService_Delete(t *testing.T) {
	svc, ctx := newTestService(t)
	c, err := svc.Create(ctx, CreateInput{Name: "Silinecek", Axis: AxisType})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(ctx, c.ID))

	_, err = svc.GetPublicBySlug(ctx, "silinecek")
	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestService_Delete_NotFound(t *testing.T) {
	svc, ctx := newTestService(t)

	err := svc.Delete(ctx, 9999)

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestService_GetPublicBySlug_HidesInactive(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{Name: "Pasif", Axis: AxisType, IsActive: false})
	require.NoError(t, err)

	_, err = svc.GetPublicBySlug(ctx, "pasif")

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestService_ProductCount_Empty(t *testing.T) {
	svc, ctx := newTestService(t)
	c, err := svc.Create(ctx, CreateInput{Name: "Boş", Axis: AxisType})
	require.NoError(t, err)

	count, err := svc.ProductCount(ctx, c.ID)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// Olmayan kategori için count(*) 0 döner (aggregate) ama bu yanıltıcıdır —
// service GetByID ile önce kategorinin var olduğunu doğrulamalı.
func TestService_ProductCount_NotFound(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.ProductCount(ctx, 9999)

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}
