package database

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// NewTestDB test veritabanına bağlanır ve tüm tabloları temizler.
// TEST_DATABASE_URL yoksa test skip edilir.
func NewTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://cicekci:cicekci@localhost:5434/cicekci_test?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := Connect(ctx, url)
	if err != nil {
		t.Skipf("test DB yok, skip: %v (make db-up çalıştırdın mı?)", err)
	}

	truncateAll(t, pool)

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func truncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE products, product_slugs, product_images,
		         categories, product_categories, admin_users
		RESTART IDENTITY CASCADE
	`)
	require.NoError(t, err, "test DB temizlenemedi — migration çalıştı mı?")
}
