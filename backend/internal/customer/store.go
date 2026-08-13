package customer

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Create yeni müşteri kaydeder. passwordHash zaten hashlenmiş olmalı.
// Email çakışması (UNIQUE) → ErrConflict.
func (s *Store) Create(ctx context.Context, email, passwordHash, name, phone string) (*Customer, error) {
	var cst Customer
	err := s.pool.QueryRow(ctx,
		`INSERT INTO customers (email, password_hash, name, phone)
		 VALUES ($1,$2,$3,$4)
		 RETURNING id, email, name, phone, password_hash, created_at, updated_at`,
		email, passwordHash, name, phone,
	).Scan(&cst.ID, &cst.Email, &cst.Name, &cst.Phone, &cst.PasswordHash, &cst.CreatedAt, &cst.UpdatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, errorsx.ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("müşteri oluştur: %w", err)
	}
	return &cst, nil
}

func (s *Store) FindByEmail(ctx context.Context, email string) (*Customer, error) {
	var cst Customer
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, phone, password_hash, created_at, updated_at FROM customers WHERE email = $1`,
		email,
	).Scan(&cst.ID, &cst.Email, &cst.Name, &cst.Phone, &cst.PasswordHash, &cst.CreatedAt, &cst.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("müşteri ara: %w", err)
	}
	return &cst, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Customer, error) {
	var cst Customer
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, phone, password_hash, created_at, updated_at FROM customers WHERE id = $1`,
		id,
	).Scan(&cst.ID, &cst.Email, &cst.Name, &cst.Phone, &cst.PasswordHash, &cst.CreatedAt, &cst.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("müşteri getir: %w", err)
	}
	return &cst, nil
}

func (s *Store) UpdateProfile(ctx context.Context, id int64, name, phone string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE customers SET name=$2, phone=$3, updated_at=now() WHERE id=$1`,
		id, name, phone)
	if err != nil {
		return fmt.Errorf("profil güncelle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

func (s *Store) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE customers SET password_hash=$2, updated_at=now() WHERE id=$1`,
		id, passwordHash)
	if err != nil {
		return fmt.Errorf("şifre güncelle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

// List müşterileri en yeni kayıttan eskiye listeler (admin — salt okunur).
// q boşsa tüm müşteriler; doluysa email VEYA name içinde arama (ILIKE).
// PasswordHash bilerek SEÇİLMİYOR — listeleme ekranının buna ihtiyacı yok.
func (s *Store) List(ctx context.Context, q string, limit, offset int) ([]Customer, error) {
	const baseSelect = `SELECT id, email, name, phone, created_at, updated_at FROM customers`

	query := baseSelect
	args := []any{}
	if q != "" {
		query += ` WHERE email ILIKE $1 OR name ILIKE $1`
		args = append(args, "%"+q+"%")
	}
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("müşteri listele: %w", err)
	}
	defer rows.Close()

	customers := []Customer{}
	for rows.Next() {
		var cst Customer
		if err := rows.Scan(&cst.ID, &cst.Email, &cst.Name, &cst.Phone, &cst.CreatedAt, &cst.UpdatedAt); err != nil {
			return nil, err
		}
		customers = append(customers, cst)
	}

	return customers, rows.Err()
}

// Count List ile aynı filtreyle toplam müşteri sayısını döner (sayfalama için).
func (s *Store) Count(ctx context.Context, q string) (int, error) {
	query := `SELECT count(*) FROM customers`
	args := []any{}
	if q != "" {
		query += ` WHERE email ILIKE $1 OR name ILIKE $1`
		args = append(args, "%"+q+"%")
	}

	var total int
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("müşteri say: %w", err)
	}
	return total, nil
}
