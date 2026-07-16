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
