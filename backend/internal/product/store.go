package product

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

// productSelect ürünü güncel slug'ı ve kategori id'leriyle birlikte çeker.
const productSelect = `
	SELECT p.id, p.name,
	       COALESCE(ps.slug, ''),
	       p.description, p.price, p.is_active, p.is_featured,
	       COALESCE(
	         (SELECT array_agg(pc.category_id ORDER BY pc.category_id)
	          FROM product_categories pc WHERE pc.product_id = p.id),
	         '{}'
	       ),
	       p.created_at, p.updated_at
	FROM products p
	LEFT JOIN product_slugs ps ON ps.product_id = p.id AND ps.is_current
`

func scanProduct(row pgx.Row) (*Product, error) {
	var p Product
	err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price,
		&p.IsActive, &p.IsFeatured, &p.CategoryIDs, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("ürün scan: %w", err)
	}
	return &p, nil
}

// Create ürünü ve ilk slug'ını tek transaction'da yazar.
func (s *Store) Create(ctx context.Context, in CreateInput, slug string) (*Product, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO products (name, description, price, is_active, is_featured)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		in.Name, in.Description, in.Price, in.IsActive, in.IsFeatured,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("ürün ekle: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO product_slugs (slug, product_id, is_current) VALUES ($1, $2, true)`,
		slug, id)
	if err != nil {
		return nil, fmt.Errorf("slug ekle: %w", err)
	}

	if len(in.CategoryIDs) > 0 {
		if err := insertCategories(ctx, tx, id, in.CategoryIDs); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx commit: %w", err)
	}

	return s.GetByID(ctx, id)
}

func insertCategories(ctx context.Context, tx pgx.Tx, productID int64, categoryIDs []int64) error {
	for _, cid := range categoryIDs {
		_, err := tx.Exec(ctx,
			`INSERT INTO product_categories (product_id, category_id) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`,
			productID, cid)
		if err != nil {
			return fmt.Errorf("kategori bağla: %w", err)
		}
	}
	return nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Product, error) {
	return scanProduct(s.pool.QueryRow(ctx, productSelect+` WHERE p.id = $1`, id))
}

// GetPublicByID sadece aktif ürünü döner — is_active filtresi store'da (spec §4.6).
func (s *Store) GetPublicByID(ctx context.Context, id int64) (*Product, error) {
	return scanProduct(s.pool.QueryRow(ctx,
		productSelect+` WHERE p.id = $1 AND p.is_active`, id))
}

// FindSlug slug'ın hangi ürüne ait olduğunu ve güncel olup olmadığını döner.
// isCurrent=false ise handler 301 redirect yapmalı (spec §4.2).
func (s *Store) FindSlug(ctx context.Context, slug string) (int64, bool, error) {
	var productID int64
	var isCurrent bool
	err := s.pool.QueryRow(ctx,
		`SELECT product_id, is_current FROM product_slugs WHERE slug = $1`, slug,
	).Scan(&productID, &isCurrent)

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, errorsx.ErrNotFound
	}
	if err != nil {
		return 0, false, fmt.Errorf("slug ara: %w", err)
	}
	return productID, isCurrent, nil
}

// AddSlug yeni slug ekler ve eskisini is_current=false yapar.
// Partial unique index bir üründe tek güncel slug garantiler.
func (s *Store) AddSlug(ctx context.Context, productID int64, slug string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE product_slugs SET is_current = false
		 WHERE product_id = $1 AND is_current`, productID)
	if err != nil {
		return fmt.Errorf("eski slug pasifle: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO product_slugs (slug, product_id, is_current) VALUES ($1, $2, true)`,
		slug, productID)
	if err != nil {
		return fmt.Errorf("yeni slug ekle: %w", err)
	}

	return tx.Commit(ctx)
}

// SlugExists eski slug'ları da kapsar — bir slug bir kez kullanıldıysa
// başka ürüne verilemez.
func (s *Store) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM product_slugs WHERE slug = $1)`, slug,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("slug kontrol: %w", err)
	}
	return exists, nil
}

// Update ürünü günceller. newSlug boş değilse slug geçmişi de AYNI
// transaction'da güncellenir: eski slug is_current=false olur, yenisi
// eklenir. Böylece isim ve slug asla birbirinden ayrı düşmez.
func (s *Store) Update(ctx context.Context, id int64, in UpdateInput, newSlug string) (*Product, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx,
		`UPDATE products SET
		   name        = COALESCE($2, name),
		   description = COALESCE($3, description),
		   price       = COALESCE($4, price),
		   is_active   = COALESCE($5, is_active),
		   is_featured = COALESCE($6, is_featured),
		   updated_at  = now()
		 WHERE id = $1`,
		id, in.Name, in.Description, in.Price, in.IsActive, in.IsFeatured)
	if err != nil {
		return nil, fmt.Errorf("ürün güncelle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, errorsx.ErrNotFound
	}

	// CategoryIDs nil ise dokunma; boş slice ise hepsini kaldır.
	if in.CategoryIDs != nil {
		if _, err := tx.Exec(ctx,
			`DELETE FROM product_categories WHERE product_id = $1`, id); err != nil {
			return nil, fmt.Errorf("kategori temizle: %w", err)
		}
		if err := insertCategories(ctx, tx, id, in.CategoryIDs); err != nil {
			return nil, err
		}
	}

	// newSlug doluysa slug geçmişi de aynı tx'te güncellenir — ürün adı
	// ile slug'ın birbirinden ayrı düşmesi (biri commit olup diğeri
	// olmaması) bu sayede imkânsız hale gelir.
	if newSlug != "" {
		if _, err := tx.Exec(ctx,
			`UPDATE product_slugs SET is_current = false
			 WHERE product_id = $1 AND is_current`, id); err != nil {
			return nil, fmt.Errorf("eski slug pasifle: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_slugs (slug, product_id, is_current) VALUES ($1, $2, true)`,
			newSlug, id); err != nil {
			return nil, fmt.Errorf("yeni slug ekle: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx commit: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("ürün sil: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

// SetCategories ürünün kategorilerini tamamen değiştirir.
func (s *Store) SetCategories(ctx context.Context, productID int64, categoryIDs []int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tx başlat: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM product_categories WHERE product_id = $1`, productID); err != nil {
		return fmt.Errorf("kategori temizle: %w", err)
	}

	if err := insertCategories(ctx, tx, productID, categoryIDs); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ListPublic aktif ürünleri filtreyle listeler.
// İki eksen de doluysa AND — her iki koşula da uyan ürünler (spec §5.6).
// Pasif kategoriler filtrede eşleşmez.
func (s *Store) ListPublic(ctx context.Context, f Filter) ([]Product, error) {
	query := productSelect + ` WHERE p.is_active`
	args := []any{}
	argN := 1

	// Sabit koşul, argüman almıyor — ana sayfa vitrini için.
	if f.FeaturedOnly {
		query += ` AND p.is_featured`
	}

	if f.OccasionSlug != nil {
		query += fmt.Sprintf(`
			AND EXISTS (
				SELECT 1 FROM product_categories pc
				JOIN categories c ON c.id = pc.category_id
				WHERE pc.product_id = p.id
				  AND c.slug = $%d AND c.axis = 'occasion' AND c.is_active
			)`, argN)
		args = append(args, *f.OccasionSlug)
		argN++
	}

	if f.TypeSlug != nil {
		query += fmt.Sprintf(`
			AND EXISTS (
				SELECT 1 FROM product_categories pc
				JOIN categories c ON c.id = pc.category_id
				WHERE pc.product_id = p.id
				  AND c.slug = $%d AND c.axis = 'type' AND c.is_active
			)`, argN)
		args = append(args, *f.TypeSlug)
		argN++
	}

	if f.Query != nil {
		query += fmt.Sprintf(`
			AND (
				p.name ILIKE $%d
				OR EXISTS (
					SELECT 1 FROM product_categories pc
					JOIN categories c ON c.id = pc.category_id
					WHERE pc.product_id = p.id
					  AND c.is_active AND c.name ILIKE $%d
				)
			)`, argN, argN)
		args = append(args, "%"+*f.Query+"%")
		argN++
	}

	query += fmt.Sprintf(` ORDER BY p.created_at DESC, p.id DESC LIMIT $%d OFFSET $%d`,
		argN, argN+1)
	args = append(args, f.Limit, f.Offset)

	return s.queryProducts(ctx, query, args...)
}

func (s *Store) ListAdmin(ctx context.Context, limit, offset int) ([]Product, error) {
	return s.queryProducts(ctx,
		productSelect+` ORDER BY p.created_at DESC, p.id DESC LIMIT $1 OFFSET $2`,
		limit, offset)
}

func (s *Store) queryProducts(ctx context.Context, query string, args ...any) ([]Product, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ürün listele: %w", err)
	}
	defer rows.Close()

	out := make([]Product, 0)
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price,
			&p.IsActive, &p.IsFeatured, &p.CategoryIDs, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ürün scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
