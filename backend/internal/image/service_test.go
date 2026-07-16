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
// kaydeder ve istenen çağrıda hata döndürebilir. Atomiklik testleri için:
// gerçek bir saklama hatasını (disk dolu, R2 erişilemez) simüle etmenin
// başka yolu yok.
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

func (f *fakeStore) Put(ctx context.Context, prefix Prefix, key string, size Size, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.putCount++
	if f.failPutN > 0 && f.putCount == f.failPutN {
		return errors.New("simüle edilmiş saklama hatası")
	}
	f.put[objectPath(prefix, key, size)] = data
	return nil
}

func (f *fakeStore) Delete(ctx context.Context, prefix Prefix, key string, sizes []Size) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = append(f.deleted, key)
	for _, size := range sizes {
		delete(f.put, objectPath(prefix, key, size))
	}
	return nil
}

func (f *fakeStore) URL(prefix Prefix, key string, size Size) string {
	return "https://cdn.test/" + objectPath(prefix, key, size)
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

func TestService_Upload_StoredDataIsJPEG(t *testing.T) {
	svc, store, pool, ctx := newTestService(t)
	pid := insertProduct(t, pool, "Buket")

	img, err := svc.Upload(ctx, pid, makeJPEG(t, 800, 600))
	require.NoError(t, err)

	stored := store.put[objectPath(PrefixProducts, img.ImageKey, Size400)]
	require.NotEmpty(t, stored)
	format, err := DetectFormat(stored)
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)
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

// Olmayan ürüne yükleme: DB FK hatası verir, dosyalar temizlenmeli.
func TestService_Upload_DBFails_CleansUpFiles(t *testing.T) {
	svc, store, _, ctx := newTestService(t)

	_, err := svc.Upload(ctx, 9999, makeJPEG(t, 800, 600))

	require.Error(t, err)
	assert.Equal(t, 0, store.storedCount(), "DB hatası sonrası dosyalar temizlenmeli")
	assert.NotEmpty(t, store.deleted, "temizlik için Delete çağrılmalı")
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

// Ürün silme akışı: key'ler önce okunur, sonra saklama temizlenir (spec §4.4).
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

	assert.Equal(t, "https://cdn.test/products/abc123/400.jpg", url)
}
