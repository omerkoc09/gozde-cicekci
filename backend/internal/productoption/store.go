package productoption

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

const groupColumns = `id, name, slug, kind, sort_order, is_active`
const valueColumns = `id, group_id, name, swatch_hex, sort_order, is_active`

func scanGroup(row pgx.Row) (*Group, error) {
	var g Group
	err := row.Scan(&g.ID, &g.Name, &g.Slug, &g.Kind, &g.SortOrder, &g.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("seçenek grubu scan: %w", err)
	}
	g.Values = []Value{}
	return &g, nil
}

func (s *Store) CreateGroup(ctx context.Context, in CreateGroupInput, slug string) (*Group, error) {
	max, err := s.MaxGroupSortOrder(ctx)
	if err != nil {
		return nil, err
	}
	return scanGroup(s.pool.QueryRow(ctx,
		`INSERT INTO option_groups (name, slug, kind, sort_order)
		 VALUES ($1, $2, $3, $4) RETURNING `+groupColumns,
		in.Name, slug, in.Kind, max+1,
	))
}

// GroupSlugExists çakışma kontrolü — service -2, -3 eki eklerken kullanır.
func (s *Store) GroupSlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM option_groups WHERE slug = $1)`, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("slug kontrol: %w", err)
	}
	return exists, nil
}

// MaxGroupSortOrder yeni grubu sona eklemek için. Hiç grup yoksa -1.
func (s *Store) MaxGroupSortOrder(ctx context.Context) (int, error) {
	var max int
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(max(sort_order), -1) FROM option_groups`).Scan(&max)
	if err != nil {
		return 0, fmt.Errorf("sıra oku: %w", err)
	}
	return max, nil
}

// GetGroup grubu değerleriyle birlikte döner.
func (s *Store) GetGroup(ctx context.Context, id int64) (*Group, error) {
	g, err := scanGroup(s.pool.QueryRow(ctx,
		`SELECT `+groupColumns+` FROM option_groups WHERE id = $1`, id))
	if err != nil {
		return nil, err
	}

	byGroup, err := s.valuesOfMany(ctx, []int64{id}, false)
	if err != nil {
		return nil, err
	}
	if v, ok := byGroup[id]; ok {
		g.Values = v
	}
	return g, nil
}

func (s *Store) UpdateGroup(ctx context.Context, id int64, in UpdateGroupInput) (*Group, error) {
	return scanGroup(s.pool.QueryRow(ctx,
		`UPDATE option_groups SET
		   name      = COALESCE($2, name),
		   is_active = COALESCE($3, is_active)
		 WHERE id = $1
		 RETURNING `+groupColumns,
		id, in.Name, in.IsActive,
	))
}

func (s *Store) DeleteGroup(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM option_groups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("seçenek grubu sil: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

// ListGroups grupları değerleriyle döner. onlyActive true ise hem pasif
// gruplar hem pasif değerler elenir (public akış).
//
// Değerler TEK batch sorguyla çekilir — grup başına ayrı sorgu açmak
// N+1 olurdu (order.Store.List'te aynı ders alınmıştı).
func (s *Store) ListGroups(ctx context.Context, onlyActive bool) ([]Group, error) {
	where := ``
	if onlyActive {
		where = ` WHERE is_active`
	}

	rows, err := s.pool.Query(ctx,
		`SELECT `+groupColumns+` FROM option_groups`+where+` ORDER BY sort_order, id`)
	if err != nil {
		return nil, fmt.Errorf("seçenek grupları listele: %w", err)
	}
	defer rows.Close()

	groups := make([]Group, 0)
	ids := make([]int64, 0)
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Slug, &g.Kind, &g.SortOrder, &g.IsActive); err != nil {
			return nil, fmt.Errorf("seçenek grubu scan: %w", err)
		}
		g.Values = []Value{}
		groups = append(groups, g)
		ids = append(ids, g.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return groups, nil
	}

	byGroup, err := s.valuesOfMany(ctx, ids, onlyActive)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if v, ok := byGroup[groups[i].ID]; ok {
			groups[i].Values = v
		}
	}
	return groups, nil
}

// valuesOfMany birden çok grubun değerlerini tek sorguda çeker.
func (s *Store) valuesOfMany(ctx context.Context, groupIDs []int64, onlyActive bool) (map[int64][]Value, error) {
	where := `WHERE group_id = ANY($1)`
	if onlyActive {
		where += ` AND is_active`
	}

	rows, err := s.pool.Query(ctx,
		`SELECT `+valueColumns+` FROM option_values `+where+` ORDER BY sort_order, id`, groupIDs)
	if err != nil {
		return nil, fmt.Errorf("seçenek değerleri listele: %w", err)
	}
	defer rows.Close()

	out := make(map[int64][]Value, len(groupIDs))
	for rows.Next() {
		var v Value
		if err := rows.Scan(&v.ID, &v.GroupID, &v.Name, &v.SwatchHex, &v.SortOrder, &v.IsActive); err != nil {
			return nil, fmt.Errorf("seçenek değeri scan: %w", err)
		}
		out[v.GroupID] = append(out[v.GroupID], v)
	}
	return out, rows.Err()
}

// CreateValue değeri grubun sonuna ekler.
func (s *Store) CreateValue(ctx context.Context, in CreateValueInput) (*Value, error) {
	var max int
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(max(sort_order), -1) FROM option_values WHERE group_id = $1`,
		in.GroupID).Scan(&max)
	if err != nil {
		return nil, fmt.Errorf("sıra oku: %w", err)
	}

	return scanValue(s.pool.QueryRow(ctx,
		`INSERT INTO option_values (group_id, name, swatch_hex, sort_order)
		 VALUES ($1, $2, $3, $4) RETURNING `+valueColumns,
		in.GroupID, in.Name, in.SwatchHex, max+1,
	))
}

func scanValue(row pgx.Row) (*Value, error) {
	var v Value
	err := row.Scan(&v.ID, &v.GroupID, &v.Name, &v.SwatchHex, &v.SortOrder, &v.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("seçenek değeri scan: %w", err)
	}
	return &v, nil
}

func (s *Store) GetValue(ctx context.Context, id int64) (*Value, error) {
	return scanValue(s.pool.QueryRow(ctx,
		`SELECT `+valueColumns+` FROM option_values WHERE id = $1`, id))
}

func (s *Store) UpdateValue(ctx context.Context, id int64, in UpdateValueInput) (*Value, error) {
	return scanValue(s.pool.QueryRow(ctx,
		`UPDATE option_values SET
		   name       = COALESCE($2, name),
		   swatch_hex = COALESCE($3, swatch_hex),
		   is_active  = COALESCE($4, is_active)
		 WHERE id = $1
		 RETURNING `+valueColumns,
		id, in.Name, in.SwatchHex, in.IsActive,
	))
}

func (s *Store) DeleteValue(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM option_values WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("seçenek değeri sil: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}
