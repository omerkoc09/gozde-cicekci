package productoption

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/pkg/errorsx"
)

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

// hexPattern "#RRGGBB" — kısa form (#FFF) kabul edilmiyor: tek biçim
// tutmak panelde ve müşteri sayfasında sürprizi önler.
var hexPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func (s *Service) CreateGroup(ctx context.Context, in CreateGroupInput) (*Group, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, fmt.Errorf("%w: seçenek grubu adı boş olamaz", errorsx.ErrInvalidInput)
	}
	if !in.Kind.Valid() {
		return nil, fmt.Errorf("%w: geçersiz seçenek tipi %q (color veya text olmalı)",
			errorsx.ErrInvalidInput, in.Kind)
	}

	slug, err := s.uniqueSlug(ctx, product.Slugify(in.Name))
	if err != nil {
		return nil, err
	}
	return s.store.CreateGroup(ctx, in, slug)
}

func (s *Service) uniqueSlug(ctx context.Context, base string) (string, error) {
	candidate := base
	for i := 2; ; i++ {
		exists, err := s.store.GroupSlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

// UpdateGroup adı ve aktifliği günceller. Kind ve slug DEĞİŞMEZ.
func (s *Service) UpdateGroup(ctx context.Context, id int64, in UpdateGroupInput) (*Group, error) {
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: seçenek grubu adı boş olamaz", errorsx.ErrInvalidInput)
		}
		in.Name = &trimmed
	}
	return s.store.UpdateGroup(ctx, id, in)
}

func (s *Service) DeleteGroup(ctx context.Context, id int64) error {
	return s.store.DeleteGroup(ctx, id)
}

func (s *Service) ListGroups(ctx context.Context, onlyActive bool) ([]Group, error) {
	return s.store.ListGroups(ctx, onlyActive)
}

func (s *Service) GetGroup(ctx context.Context, id int64) (*Group, error) {
	return s.store.GetGroup(ctx, id)
}

// GroupProductCount silme öncesi uyarı için. Grup yoksa ErrNotFound —
// count(*) aggregate olduğu için store tek başına ayırt edemez.
func (s *Service) GroupProductCount(ctx context.Context, groupID int64) (int, error) {
	if _, err := s.store.GetGroup(ctx, groupID); err != nil {
		return 0, err
	}
	return s.store.GroupProductCount(ctx, groupID)
}

// ProductsUsingGroup grubu kullanan ürünleri döner — panelde "bu grup
// hangi ürünlerde soruluyor" listesi. Grup yoksa ErrNotFound: boş liste
// ile "grup yok" durumunu ayırt edebilmek için.
func (s *Service) ProductsUsingGroup(ctx context.Context, groupID int64) ([]GroupProduct, error) {
	if _, err := s.store.GetGroup(ctx, groupID); err != nil {
		return nil, err
	}
	return s.store.ProductsUsingGroup(ctx, groupID)
}

// CreateValue değeri gruba ekler. Renk grubunda hex zorunlu ve
// "#RRGGBB" formatında; metin grubunda hex temizlenir.
func (s *Service) CreateValue(ctx context.Context, in CreateValueInput) (*Value, error) {
	g, err := s.store.GetGroup(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}

	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, fmt.Errorf("%w: seçenek adı boş olamaz", errorsx.ErrInvalidInput)
	}

	in.SwatchHex, err = normalizeHex(g.Kind, in.SwatchHex)
	if err != nil {
		return nil, err
	}

	return s.store.CreateValue(ctx, in)
}

func (s *Service) UpdateValue(ctx context.Context, id int64, in UpdateValueInput) (*Value, error) {
	cur, err := s.store.GetValue(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: seçenek adı boş olamaz", errorsx.ErrInvalidInput)
		}
		in.Name = &trimmed
	}

	if in.SwatchHex != nil {
		g, err := s.store.GetGroup(ctx, cur.GroupID)
		if err != nil {
			return nil, err
		}
		hex, err := normalizeHex(g.Kind, *in.SwatchHex)
		if err != nil {
			return nil, err
		}
		in.SwatchHex = &hex
	}

	return s.store.UpdateValue(ctx, id, in)
}

func (s *Service) DeleteValue(ctx context.Context, id int64) error {
	return s.store.DeleteValue(ctx, id)
}

// normalizeHex renk grubunda hex'i doğrular, metin grubunda temizler.
func normalizeHex(kind Kind, hex string) (string, error) {
	hex = strings.TrimSpace(hex)

	if kind == KindText {
		return "", nil // metin grubunda renk saklanmaz
	}

	if hex == "" {
		return "", fmt.Errorf("%w: renk seçeneğinde renk kodu zorunlu", errorsx.ErrInvalidInput)
	}
	if !hexPattern.MatchString(hex) {
		return "", fmt.Errorf("%w: geçersiz renk kodu %q (#RRGGBB olmalı, örn. #F0A6CA)",
			errorsx.ErrInvalidInput, hex)
	}
	return strings.ToUpper(hex), nil
}

// ReorderGroups grupları ids sırasına dizer. ids TÜM grupları içermeli —
// kısmi sıralama listede olmayan grupları sessizce 0'a düşürüp sırayı bozar.
func (s *Service) ReorderGroups(ctx context.Context, ids []int64) error {
	mevcut, err := s.store.GroupIDs(ctx)
	if err != nil {
		return err
	}
	if err := dogrulaSiralama(mevcut, ids, "grup"); err != nil {
		return err
	}
	return s.store.ReorderGroups(ctx, ids)
}

// ReorderValues gruptaki değerleri ids sırasına dizer. ids O GRUBUN tüm
// değerlerini içermeli.
func (s *Service) ReorderValues(ctx context.Context, groupID int64, ids []int64) error {
	mevcut, err := s.store.ValueIDsOfGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if err := dogrulaSiralama(mevcut, ids, "seçenek"); err != nil {
		return err
	}
	return s.store.ReorderValues(ctx, ids)
}

// dogrulaSiralama gelen listenin mevcutların TAM bir permütasyonu
// olduğunu doğrular: eksik, fazla, tekrarlı veya yabancı ID reddedilir.
func dogrulaSiralama(mevcut, gelen []int64, ad string) error {
	if len(gelen) != len(mevcut) {
		return fmt.Errorf("%w: sıralama listesi tüm %sları içermeli (%d bekleniyordu, %d geldi)",
			errorsx.ErrInvalidInput, ad, len(mevcut), len(gelen))
	}

	beklenen := make(map[int64]bool, len(mevcut))
	for _, id := range mevcut {
		beklenen[id] = true
	}

	gorulen := make(map[int64]bool, len(gelen))
	for _, id := range gelen {
		if gorulen[id] {
			return fmt.Errorf("%w: sıralama listesinde tekrar eden %s", errorsx.ErrInvalidInput, ad)
		}
		if !beklenen[id] {
			return fmt.Errorf("%w: sıralama listesinde buraya ait olmayan %s", errorsx.ErrInvalidInput, ad)
		}
		gorulen[id] = true
	}
	return nil
}

// ValidateGroupLinks links listesinin geçerliliğini DB'ye yazmadan kontrol
// eder: aynı grup iki kez gelmemeli, her grup ID'si var olmalı. Handler'lar
// ürünü kaydetmeden ÖNCE bunu çağırarak "ürün kaydedildi ama grup bağlama
// başarısız oldu" kısmi başarısını önleyebilir — SetProductGroups zaten
// aynı kontrolü yapıyor, kod tekrarını önlemek için oradan da çağrılıyor.
func (s *Service) ValidateGroupLinks(ctx context.Context, links []ProductGroupLink) error {
	gorulen := make(map[int64]bool, len(links))
	for _, l := range links {
		if gorulen[l.GroupID] {
			return fmt.Errorf("%w: aynı seçenek grubu iki kez gönderildi", errorsx.ErrInvalidInput)
		}
		gorulen[l.GroupID] = true

		if _, err := s.store.GetGroup(ctx, l.GroupID); err != nil {
			return fmt.Errorf("%w: seçenek grubu bulunamadı", errorsx.ErrInvalidInput)
		}
	}
	return nil
}

// SetProductGroups ürünün seçenek gruplarını komple değiştirir.
func (s *Service) SetProductGroups(ctx context.Context, productID int64, links []ProductGroupLink) error {
	if err := s.ValidateGroupLinks(ctx, links); err != nil {
		return err
	}
	return s.store.SetProductGroups(ctx, productID, links)
}

func (s *Service) GroupsForProduct(ctx context.Context, productID int64, onlyActive bool) ([]ProductGroup, error) {
	return s.store.GroupsForProduct(ctx, productID, onlyActive)
}
