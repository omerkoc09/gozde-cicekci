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

// GroupIDs tüm grup ID'lerini döner — ReorderGroups'un gelen listeyi
// mevcutların tamamıyla karşılaştırması için.
func (s *Store) GroupIDs(ctx context.Context) ([]int64, error) {
	return s.scanIDs(ctx, `SELECT id FROM option_groups`)
}

// ValueIDsOfGroup gruptaki tüm değer ID'lerini döner.
func (s *Store) ValueIDsOfGroup(ctx context.Context, groupID int64) ([]int64, error) {
	return s.scanIDs(ctx, `SELECT id FROM option_values WHERE group_id = $1`, groupID)
}

func (s *Store) scanIDs(ctx context.Context, query string, args ...any) ([]int64, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("id listele: %w", err)
	}
	defer rows.Close()

	out := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("id scan: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ReorderGroups ids sırasına göre sort_order'ı 0,1,2... yazar.
// Tek transaction: yarım kalırsa hiçbiri uygulanmaz.
func (s *Store) ReorderGroups(ctx context.Context, ids []int64) error {
	return s.reorder(ctx, `UPDATE option_groups SET sort_order = $2 WHERE id = $1`, ids)
}

func (s *Store) ReorderValues(ctx context.Context, ids []int64) error {
	return s.reorder(ctx, `UPDATE option_values SET sort_order = $2 WHERE id = $1`, ids)
}

func (s *Store) reorder(ctx context.Context, query string, ids []int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("sıralama tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for i, id := range ids {
		if _, err := tx.Exec(ctx, query, id, i); err != nil {
			return fmt.Errorf("sıra yaz: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("sıralama commit: %w", err)
	}
	return nil
}

// GroupProductCount grubu kaç ürünün kullandığı — silme öncesi uyarı için.
func (s *Store) GroupProductCount(ctx context.Context, groupID int64) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM product_option_groups WHERE group_id = $1`, groupID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("ürün say: %w", err)
	}
	return n, nil
}

// SetProductGroups ürünün seçenek gruplarını KOMPLE değiştirir (önce siler,
// sonra yazar) — tek transaction. Boş links tüm bağları kaldırır.
func (s *Store) SetProductGroups(ctx context.Context, productID int64, links []ProductGroupLink) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ürün seçenek tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM product_option_groups WHERE product_id = $1`, productID); err != nil {
		return fmt.Errorf("ürün seçenek sil: %w", err)
	}

	for _, l := range links {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_option_groups (product_id, group_id, is_required)
			 VALUES ($1, $2, $3)`, productID, l.GroupID, l.IsRequired); err != nil {
			return fmt.Errorf("ürün seçenek ekle: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ürün seçenek commit: %w", err)
	}
	return nil
}

// GroupsForProduct ürüne açık grupları değerleriyle döner.
// onlyActive true ise pasif grup/değer elenir (public akış).
func (s *Store) GroupsForProduct(ctx context.Context, productID int64, onlyActive bool) ([]ProductGroup, error) {
	where := `WHERE pog.product_id = $1`
	if onlyActive {
		where += ` AND g.is_active`
	}

	rows, err := s.pool.Query(ctx,
		`SELECT g.id, g.name, g.slug, g.kind, g.sort_order, g.is_active, pog.is_required
		 FROM product_option_groups pog
		 JOIN option_groups g ON g.id = pog.group_id `+where+`
		 ORDER BY g.sort_order, g.id`, productID)
	if err != nil {
		return nil, fmt.Errorf("ürün seçenek grupları: %w", err)
	}
	defer rows.Close()

	out := make([]ProductGroup, 0)
	ids := make([]int64, 0)
	for rows.Next() {
		var pg ProductGroup
		if err := rows.Scan(&pg.ID, &pg.Name, &pg.Slug, &pg.Kind,
			&pg.SortOrder, &pg.IsActive, &pg.IsRequired); err != nil {
			return nil, fmt.Errorf("ürün seçenek grubu scan: %w", err)
		}
		pg.Values = []Value{}
		out = append(out, pg)
		ids = append(ids, pg.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return out, nil
	}

	byGroup, err := s.valuesOfMany(ctx, ids, onlyActive)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if v, ok := byGroup[out[i].ID]; ok {
			out[i].Values = v
		}
	}
	return out, nil
}
