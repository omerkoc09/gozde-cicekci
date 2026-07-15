package image

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalStore görselleri dosya sistemine yazar. Geliştirme ve testte kullanılır.
// Production'da R2Store kullanılır — ikisi de Store interface'ini uygular,
// uygulama kodu hangisi olduğunu bilmez (spec §4.4).
type LocalStore struct {
	baseDir string
	baseURL string
}

func NewLocalStore(baseDir, baseURL string) (*LocalStore, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("baseDir boş olamaz")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("upload dizini oluştur: %w", err)
	}
	return &LocalStore{
		baseDir: baseDir,
		baseURL: strings.TrimRight(baseURL, "/"),
	}, nil
}

// safeKey key'in dizin dışına çıkmadığını doğrular.
func safeKey(key string) error {
	if key == "" {
		return fmt.Errorf("key boş olamaz")
	}
	if strings.Contains(key, "/") || strings.Contains(key, "\\") || strings.Contains(key, "..") {
		return fmt.Errorf("geçersiz key: %q", key)
	}
	return nil
}

func (s *LocalStore) Put(ctx context.Context, key string, size Size, data []byte) error {
	if err := safeKey(key); err != nil {
		return err
	}
	if !size.Valid() {
		return fmt.Errorf("geçersiz boyut: %q", size)
	}

	full := filepath.Join(s.baseDir, objectPath(key, size))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("dizin oluştur: %w", err)
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return fmt.Errorf("dosya yaz: %w", err)
	}
	return nil
}

// Delete key'in tüm boyutlarını siler. Olmayan key hata değil — idempotent.
func (s *LocalStore) Delete(ctx context.Context, key string) error {
	if err := safeKey(key); err != nil {
		return err
	}
	dir := filepath.Join(s.baseDir, "products", key)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("dosya sil: %w", err)
	}
	return nil
}

func (s *LocalStore) URL(key string, size Size) string {
	return s.baseURL + "/" + objectPath(key, size)
}
