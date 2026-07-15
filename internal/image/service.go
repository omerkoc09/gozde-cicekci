package image

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/omerkoc/cicekci/pkg/errorsx"
)

// Service görsel işleme, saklama ve veritabanını birleştirir.
type Service struct {
	store Store
	db    *DB
}

func NewService(store Store, db *DB) *Service {
	return &Service{store: store, db: db}
}

// Upload görseli işler, tüm boyutları saklar ve DB'ye kaydeder.
//
// Atomiklik (spec §4.4): önce TÜM boyutlar saklamaya yazılır; hepsi
// başarılıysa DB'ye kayıt atılır. Ara adımda hata olursa yazılanlar silinir.
// Böylece DB kaydı olmayan dosya (yetim) veya dosyası olmayan DB kaydı
// (kırık görsel) oluşmaz.
func (s *Service) Upload(ctx context.Context, productID int64, data []byte) (*ProductImage, error) {
	processed := make(map[Size][]byte, len(AllSizes))
	for _, size := range AllSizes {
		out, err := Process(data, size)
		if err != nil {
			if errors.Is(err, ErrUnsupportedFormat) {
				return nil, fmt.Errorf("%w: %s", errorsx.ErrInvalidInput, err.Error())
			}
			return nil, fmt.Errorf("görsel işle (%s): %w", size, err)
		}
		processed[size] = out
	}

	key := NewKey()

	for _, size := range AllSizes {
		if err := s.store.Put(ctx, key, size, processed[size]); err != nil {
			// Yarım kalanı temizle — yetim dosya bırakma.
			if delErr := s.store.Delete(ctx, key); delErr != nil {
				log.Printf("yarım yükleme temizlenemedi (key=%s): %v", key, delErr)
			}
			return nil, fmt.Errorf("görsel sakla (%s): %w", size, err)
		}
	}

	img, err := s.db.Insert(ctx, productID, key)
	if err != nil {
		// DB yazılamadıysa dosyalar yetim kalır — temizle.
		if delErr := s.store.Delete(ctx, key); delErr != nil {
			log.Printf("DB hatası sonrası temizlik başarısız (key=%s): %v", key, delErr)
		}
		return nil, err
	}

	return img, nil
}

// Delete görseli DB'den ve saklamadan siler.
// Sıra önemli: key önce okunur (DB kaydı gidince öğrenilemez).
func (s *Service) Delete(ctx context.Context, imageID int64) error {
	img, err := s.db.GetByID(ctx, imageID)
	if err != nil {
		return err
	}

	if err := s.db.Delete(ctx, imageID); err != nil {
		return err
	}

	// Saklama silme başarısız olursa yetim dosya kalır ama site bozulmaz —
	// log'a düşer, kabul edilebilir (spec §4.4).
	if err := s.store.Delete(ctx, img.ImageKey); err != nil {
		log.Printf("yetim dosya: saklamadan silinemedi (key=%s): %v", img.ImageKey, err)
	}

	return nil
}

// DeleteAllForProduct ürün silinmeden ÖNCE çağrılır.
// Ürün silinince product_images CASCADE ile gider ve key'ler kaybolur —
// o yüzden dosyaları temizlemek için key'leri şimdi okumak gerekir.
func (s *Service) DeleteAllForProduct(ctx context.Context, productID int64) error {
	keys, err := s.db.KeysByProduct(ctx, productID)
	if err != nil {
		return err
	}

	for _, key := range keys {
		if err := s.store.Delete(ctx, key); err != nil {
			log.Printf("yetim dosya: saklamadan silinemedi (key=%s): %v", key, err)
		}
	}

	return nil
}

func (s *Service) ListByProduct(ctx context.Context, productID int64) ([]ProductImage, error) {
	return s.db.ListByProduct(ctx, productID)
}

func (s *Service) ListByProducts(ctx context.Context, productIDs []int64) (map[int64][]ProductImage, error) {
	return s.db.ListByProducts(ctx, productIDs)
}

func (s *Service) Reorder(ctx context.Context, productID int64, imageIDs []int64) error {
	return s.db.Reorder(ctx, productID, imageIDs)
}

// URL saklamanın ürettiği public adresi döner. Domain veritabanında değil,
// ImageStore'da yaşar (spec §4.1).
func (s *Service) URL(key string, size Size) string {
	return s.store.URL(key, size)
}
