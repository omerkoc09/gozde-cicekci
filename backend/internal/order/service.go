package order

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/shopspring/decimal"
)

// ProductReader service'in ürün okumak için ihtiyaç duyduğu tek şey.
// Dar arayüz: order paketi product'ın tamamına bağlanmasın.
type ProductReader interface {
	GetByID(ctx context.Context, id int64) (*product.Product, error)
}

// DeliveryConfig teslimat kuralları — config'den gelir (spec §4).
type DeliveryConfig struct {
	Fee           string
	Slots         []string
	SameDayCutoff string
	MaxDays       int
	// Districts teslimat yapılan ilçeler — bilgi amaçlı, ücrete etkisi yok
	// (2026-07-18 sipariş formu iyileştirmeleri spec'i).
	Districts []string

	// DistrictFees ilçeye özel teslimat ücreti ("Tire" → "80"). Listede
	// olmayan ilçe Fee'ye (genel ücret) düşer.
	DistrictFees map[string]string
}

type CreateItem struct {
	ProductID int64
	Quantity  int
}

// CreateInput handler'dan gelen ham girdi. FİYAT YOK — sunucu DB'den okur.
type CreateInput struct {
	Items []CreateItem

	BuyerName  string
	BuyerPhone string
	BuyerEmail string

	RecipientName    string
	RecipientPhone   string
	DeliveryAddress  string
	DeliveryDistrict string
	DeliveryDate     time.Time
	DeliverySlot     string
	CardMessage      string
}

type Service struct {
	store *Store
	prod  ProductReader
	cfg   DeliveryConfig
}

func NewService(store *Store, prod ProductReader, cfg DeliveryConfig) *Service {
	return &Service{store: store, prod: prod, cfg: cfg}
}

// maxQuantity absürt girdiye karşı duvar. UI'da limit YOK — 50 buket gerçek
// bir sipariş olabilir (spec §5). Bu sadece 999999999 gibi girdileri eler.
const maxQuantity = 1000

func (s *Service) Create(ctx context.Context, in CreateInput) (*Order, error) {
	if err := s.validateContact(&in); err != nil {
		return nil, err
	}
	if err := s.validateDelivery(in); err != nil {
		return nil, err
	}
	if len(in.Items) == 0 {
		return nil, fmt.Errorf("%w: sepet boş", errorsx.ErrInvalidInput)
	}

	// FİYAT DB'DEN OKUNUR — sepetten gelen fiyata asla güvenilmez (spec §2.2)
	items := make([]NewOrderItem, 0, len(in.Items))
	itemsTotal := decimal.Zero

	for _, ci := range in.Items {
		if ci.Quantity <= 0 || ci.Quantity > maxQuantity {
			return nil, fmt.Errorf("%w: geçersiz adet", errorsx.ErrInvalidInput)
		}

		p, err := s.prod.GetByID(ctx, ci.ProductID)
		if err != nil {
			return nil, fmt.Errorf("%w: ürün bulunamadı", errorsx.ErrInvalidInput)
		}
		if !p.IsActive {
			return nil, fmt.Errorf("%w: %q artık satışta değil", errorsx.ErrInvalidInput, p.Name)
		}

		// p.Price zaten decimal.Decimal — dönüşüm gerekmez
		itemsTotal = itemsTotal.Add(p.Price.Mul(decimal.NewFromInt(int64(ci.Quantity))))
		items = append(items, NewOrderItem{
			ProductID:    p.ID,
			ProductName:  p.Name,
			PriceAtOrder: p.Price,
			Quantity:     ci.Quantity,
		})
	}

	// Teslimat ücreti de siparişe kopyalanır — esnaf yarın değiştirirse
	// dünkü siparişin toplamı bozulmasın (spec §3). İlçeye özel ücret
	// varsa o kullanılır, yoksa genel ücrete düşülür.
	feeStr := s.cfg.Fee
	if districtFee, ok := s.cfg.DistrictFees[in.DeliveryDistrict]; ok {
		feeStr = districtFee
	}

	fee, err := decimal.NewFromString(feeStr)
	if err != nil {
		return nil, fmt.Errorf("teslimat ücreti okunamadı: %w", err)
	}

	return s.store.Create(ctx, NewOrder{
		BuyerName:        in.BuyerName,
		BuyerPhone:       in.BuyerPhone,
		BuyerEmail:       in.BuyerEmail,
		RecipientName:    in.RecipientName,
		RecipientPhone:   in.RecipientPhone,
		DeliveryAddress:  in.DeliveryAddress,
		DeliveryDistrict: in.DeliveryDistrict,
		DeliveryDate:     in.DeliveryDate,
		DeliverySlot:     in.DeliverySlot,
		CardMessage:      in.CardMessage,
		ItemsTotal:       itemsTotal,
		DeliveryFee:      fee,
		Total:            itemsTotal.Add(fee),
		Items:            items,
	})
}

func (s *Service) validateContact(in *CreateInput) error {
	in.BuyerName = strings.TrimSpace(in.BuyerName)
	in.BuyerPhone = strings.TrimSpace(in.BuyerPhone)
	in.BuyerEmail = strings.TrimSpace(in.BuyerEmail)
	in.RecipientName = strings.TrimSpace(in.RecipientName)
	in.RecipientPhone = strings.TrimSpace(in.RecipientPhone)
	in.DeliveryAddress = strings.TrimSpace(in.DeliveryAddress)
	in.DeliveryDistrict = strings.TrimSpace(in.DeliveryDistrict)
	in.CardMessage = strings.TrimSpace(in.CardMessage)

	switch {
	case in.BuyerName == "":
		return fmt.Errorf("%w: ad soyad gerekli", errorsx.ErrInvalidInput)
	case in.BuyerPhone == "":
		return fmt.Errorf("%w: telefon gerekli", errorsx.ErrInvalidInput)
	case in.RecipientName == "":
		return fmt.Errorf("%w: alıcı adı gerekli", errorsx.ErrInvalidInput)
	// Kurye alıcıyı arayamazsa teslimat başarısız olur (spec §3)
	case in.RecipientPhone == "":
		return fmt.Errorf("%w: alıcı telefonu gerekli", errorsx.ErrInvalidInput)
	case in.DeliveryAddress == "":
		return fmt.Errorf("%w: teslimat adresi gerekli", errorsx.ErrInvalidInput)
	case in.DeliveryDistrict == "":
		return fmt.Errorf("%w: teslimat ilçesi gerekli", errorsx.ErrInvalidInput)
	}

	return nil
}

func (s *Service) validateDelivery(in CreateInput) error {
	today := time.Now().Truncate(24 * time.Hour)
	d := in.DeliveryDate.Truncate(24 * time.Hour)

	if d.Before(today) {
		return fmt.Errorf("%w: teslimat tarihi geçmişte olamaz", errorsx.ErrInvalidInput)
	}
	if d.After(today.AddDate(0, 0, s.cfg.MaxDays)) {
		return fmt.Errorf("%w: en fazla %d gün sonrasına sipariş verilebilir",
			errorsx.ErrInvalidInput, s.cfg.MaxDays)
	}

	if !slices.Contains(s.cfg.Slots, in.DeliverySlot) {
		return fmt.Errorf("%w: geçersiz teslimat saati", errorsx.ErrInvalidInput)
	}

	if !slices.Contains(s.cfg.Districts, in.DeliveryDistrict) {
		return fmt.Errorf("%w: geçersiz teslimat ilçesi", errorsx.ErrInvalidInput)
	}

	// Aynı gün + cutoff geçmiş → esnaf yetiştiremez
	if d.Equal(today) && s.pastCutoff(time.Now()) {
		return fmt.Errorf("%w: aynı gün siparişi için saat %s'ı geçti",
			errorsx.ErrInvalidInput, s.cfg.SameDayCutoff)
	}

	return nil
}

// pastCutoff şu an cutoff saatini geçti mi. Cutoff bozuksa kısıt uygulanmaz —
// yanlış config yüzünden tüm siparişleri reddetmektense kısıtsız çalış.
func (s *Service) pastCutoff(now time.Time) bool {
	cutoff, err := time.Parse("15:04", s.cfg.SameDayCutoff)
	if err != nil {
		return false
	}
	nowMin := now.Hour()*60 + now.Minute()
	cutMin := cutoff.Hour()*60 + cutoff.Minute()

	return nowMin > cutMin
}

func (s *Service) List(ctx context.Context, status string, limit, offset int) ([]Order, error) {
	if status != "" && !Status(status).Valid() {
		return nil, fmt.Errorf("%w: geçersiz durum", errorsx.ErrInvalidInput)
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	return s.store.List(ctx, status, limit, offset)
}

func (s *Service) Get(ctx context.Context, id int64) (*Order, error) {
	return s.store.GetByID(ctx, id)
}

// Update statü ve/veya not günceller. nil alan değişmez (PATCH semantiği).
func (s *Service) Update(ctx context.Context, id int64, status *string, note *string) (*Order, error) {
	if status != nil {
		if !Status(*status).Valid() {
			return nil, fmt.Errorf("%w: geçersiz durum", errorsx.ErrInvalidInput)
		}
	}

	return s.store.Update(ctx, id, status, note)
}
