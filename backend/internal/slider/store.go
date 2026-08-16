package slider

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

const slideColumns = `id, title, subtitle, image_key, is_active, sort_order`

func scanSlide(row pgx.Row) (*Slide, error) {
	var s Slide
	err := row.Scan(&s.ID, &s.Title, &s.Subtitle, &s.ImageKey, &s.IsActive, &s.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("slayt scan: %w", err)
	}
	return &s, nil
}

// Create sort_order verilmemişse sona ekler — çağıran 0 gönderirse başa alır.
func (s *Store) Create(ctx context.Context, in CreateInput) (*Slide, error) {
	return scanSlide(s.pool.QueryRow(ctx,
		`INSERT INTO slides (title, subtitle, image_key, is_active, sort_order)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+slideColumns,
		in.Title, in.Subtitle, in.ImageKey, in.IsActive, in.SortOrder,
	))
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Slide, error) {
	return scanSlide(s.pool.QueryRow(ctx,
		`SELECT `+slideColumns+` FROM slides WHERE id = $1`, id))
}

func (s *Store) Update(ctx context.Context, id int64, in UpdateInput) (*Slide, error) {
	return scanSlide(s.pool.QueryRow(ctx,
		`UPDATE slides SET
		   title      = COALESCE($2, title),
		   subtitle   = COALESCE($3, subtitle),
		   is_active  = COALESCE($4, is_active),
		   sort_order = COALESCE($5, sort_order),
		   updated_at = now()
		 WHERE id = $1
		 RETURNING `+slideColumns,
		id, in.Title, in.Subtitle, in.IsActive, in.SortOrder,
	))
}

// UpdateImageKey görsel değiştirildiğinde çağrılır.
func (s *Store) UpdateImageKey(ctx context.Context, id int64, key string) (*Slide, error) {
	return scanSlide(s.pool.QueryRow(ctx,
		`UPDATE slides SET image_key = $2, updated_at = now()
		 WHERE id = $1
		 RETURNING `+slideColumns,
		id, key,
	))
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM slides WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("slayt sil: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

func (s *Store) list(ctx context.Context, where string) ([]Slide, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+slideColumns+` FROM slides `+where+` ORDER BY sort_order, id`)
	if err != nil {
		return nil, fmt.Errorf("slayt listele: %w", err)
	}
	defer rows.Close()

	out := make([]Slide, 0)
	for rows.Next() {
		var s Slide
		if err := rows.Scan(&s.ID, &s.Title, &s.Subtitle, &s.ImageKey,
			&s.IsActive, &s.SortOrder); err != nil {
			return nil, fmt.Errorf("slayt scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListPublic ana sayfa için — sadece aktif slaytlar.
func (s *Store) ListPublic(ctx context.Context) ([]Slide, error) {
	return s.list(ctx, `WHERE is_active`)
}

// ListAdmin pasifler dahil hepsi.
func (s *Store) ListAdmin(ctx context.Context) ([]Slide, error) {
	return s.list(ctx, ``)
}

// AllIDs tüm slayt ID'lerini döner — Reorder'ın gelen listeyi mevcutların
// tamamıyla karşılaştırması için.
func (s *Store) AllIDs(ctx context.Context) ([]int64, error) {
	rows, err := s.pool.Query(ctx, `SELECT id FROM slides`)
	if err != nil {
		return nil, fmt.Errorf("slayt id listele: %w", err)
	}
	defer rows.Close()

	out := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("slayt id scan: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// Reorder ids sırasına göre sort_order'ı 0,1,2... olarak yeniden yazar.
// Tek transaction: yarım kalırsa hiçbiri uygulanmaz.
func (s *Store) Reorder(ctx context.Context, ids []int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("sıralama tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for i, id := range ids {
		if _, err := tx.Exec(ctx,
			`UPDATE slides SET sort_order = $2, updated_at = now() WHERE id = $1`, id, i,
		); err != nil {
			return fmt.Errorf("sıra yaz: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("sıralama commit: %w", err)
	}
	return nil
}

// MaxSortOrder yeni slaytı sona eklemek için. Hiç slayt yoksa -1 döner ki
// çağıran +1 ile 0'dan başlasın.
func (s *Store) MaxSortOrder(ctx context.Context) (int, error) {
	var max int
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(max(sort_order), -1) FROM slides`).Scan(&max)
	if err != nil {
		return 0, fmt.Errorf("sıra oku: %w", err)
	}
	return max, nil
}
