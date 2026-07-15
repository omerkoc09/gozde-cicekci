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
