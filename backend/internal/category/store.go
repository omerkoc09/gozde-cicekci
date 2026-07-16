package category

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// image_key NULL olabiliyor; COALESCE ile boş string'e çeviriyoruz ki
// scan tarafında *string ile uğraşmayalım — "görsel yok" zaten boş string.
const categoryColumns = `id, name, slug, axis, is_active, is_featured, sort_order,
	COALESCE(image_key, '')`

func scanCategory(row pgx.Row) (*Category, error) {
	var c Category
	err := row.Scan(&c.ID, &c.Name, &c.Slug, &c.Axis, &c.IsActive, &c.IsFeatured,
		&c.SortOrder, &c.ImageKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("kategori scan: %w", err)
	}
	return &c, nil
}

func (s *Store) Create(ctx context.Context, in CreateInput, slug string) (*Category, error) {
	row := s.pool.QueryRow(ctx,
		`INSERT INTO categories (name, slug, axis, is_active, is_featured, sort_order)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+categoryColumns,
		in.Name, slug, in.Axis, in.IsActive, in.IsFeatured, in.SortOrder,
	)
	return scanCategory(row)
}

// SlugExists slug çakışma kontrolü için — service -2, -3 eki eklerken kullanır.
func (s *Store) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM categories WHERE slug = $1)`, slug,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("slug kontrol: %w", err)
	}
	return exists, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Category, error) {
	return scanCategory(s.pool.QueryRow(ctx,
		`SELECT `+categoryColumns+` FROM categories WHERE id = $1`, id))
}

// GetPublicBySlug sadece aktif kategoriyi döner — is_active filtresi store'da.
func (s *Store) GetPublicBySlug(ctx context.Context, slug string) (*Category, error) {
	return scanCategory(s.pool.QueryRow(ctx,
		`SELECT `+categoryColumns+` FROM categories WHERE slug = $1 AND is_active`, slug))
}

func (s *Store) Update(ctx context.Context, id int64, in UpdateInput) (*Category, error) {
	row := s.pool.QueryRow(ctx,
		`UPDATE categories SET
		   name        = COALESCE($2, name),
		   is_active   = COALESCE($3, is_active),
		   is_featured = COALESCE($4, is_featured),
		   sort_order  = COALESCE($5, sort_order)
		 WHERE id = $1
		 RETURNING `+categoryColumns,
		id, in.Name, in.IsActive, in.IsFeatured, in.SortOrder,
	)
	return scanCategory(row)
}

// UpdateImageKey kart görseli değiştiğinde çağrılır. key boş string ise
// görsel kaldırılır (NULL yazılır).
func (s *Store) UpdateImageKey(ctx context.Context, id int64, key string) (*Category, error) {
	var arg any = key
	if key == "" {
		arg = nil
	}
	return scanCategory(s.pool.QueryRow(ctx,
		`UPDATE categories SET image_key = $2 WHERE id = $1
		 RETURNING `+categoryColumns,
		id, arg,
	))
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("kategori sil: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

func (s *Store) list(ctx context.Context, where string, args ...any) ([]Category, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+categoryColumns+` FROM categories `+where+
			` ORDER BY sort_order, name`, args...)
	if err != nil {
		return nil, fmt.Errorf("kategori listele: %w", err)
	}
	defer rows.Close()

	out := make([]Category, 0)
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Axis,
			&c.IsActive, &c.IsFeatured, &c.SortOrder, &c.ImageKey); err != nil {
			return nil, fmt.Errorf("kategori scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListPublic sadece aktif kategorileri döner. axis nil ise iki eksen de gelir.
func (s *Store) ListPublic(ctx context.Context, axis *Axis) ([]Category, error) {
	if axis == nil {
		return s.list(ctx, `WHERE is_active`)
	}
	return s.list(ctx, `WHERE is_active AND axis = $1`, *axis)
}

// ListFeatured — is_active=false her şeyi ezer, featured olsa bile görünmez.
// axis nil ise iki eksen de gelir; ana sayfa "Özel Günler" (occasion) ve
// "Çiçek Türlerine Göre" (type) bölümlerini ayrı çektiği için filtreliyor.
func (s *Store) ListFeatured(ctx context.Context, axis *Axis) ([]Category, error) {
	if axis == nil {
		return s.list(ctx, `WHERE is_active AND is_featured`)
	}
	return s.list(ctx, `WHERE is_active AND is_featured AND axis = $1`, *axis)
}

func (s *Store) ListAdmin(ctx context.Context) ([]Category, error) {
	return s.list(ctx, ``)
}

func (s *Store) ProductCount(ctx context.Context, id int64) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM product_categories WHERE category_id = $1`, id,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ürün say: %w", err)
	}
	return count, nil
}
