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

// Aynı id iki kez gönderilirse reddedilir.
func TestDB_Reorder_RejectsDuplicateID(t *testing.T) {
	db, pool, ctx := newTestDB(t)
	pid := insertProduct(t, pool, "Buket")
	first, err := db.Insert(ctx, pid, "key-1")
	require.NoError(t, err)
	_, err = db.Insert(ctx, pid, "key-2")
	require.NoError(t, err)

	err = db.Reorder(ctx, pid, []int64{first.ID, first.ID})

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
