package idare

import (
	"github.com/gofiber/fiber/v2"
	"github.com/omerkoc/cicekci/internal/api"
	"github.com/omerkoc/cicekci/internal/image"
	"github.com/omerkoc/cicekci/internal/product"
	"github.com/omerkoc/cicekci/internal/productoption"
	"github.com/shopspring/decimal"
)

type productHandler struct {
	svc    *product.Service
	imgSvc *image.Service
	optSvc *productoption.Service
}

// optionGroupLinkRequest — is_required YOK. Gövdede gönderilse bile
// yok sayılır (bkz. productoption.ProductGroupLink yorumu).
type optionGroupLinkRequest struct {
	GroupID int64 `json:"group_id"`
}

func toGroupLinks(reqs []optionGroupLinkRequest) []productoption.ProductGroupLink {
	out := make([]productoption.ProductGroupLink, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, productoption.ProductGroupLink{GroupID: r.GroupID})
	}
	return out
}

type createProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       string  `json:"price"`
	IsActive    *bool   `json:"is_active"`
	IsFeatured  *bool   `json:"is_featured"`
	CategoryIDs []int64 `json:"category_ids"`
	// OptionGroups nil ise gruplar DEĞİŞMEZ; boş dizi hepsini kaldırır
	// (CategoryIDs ile aynı PATCH semantiği).
	OptionGroups []optionGroupLinkRequest `json:"option_groups"`
}

type updateProductRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Price       *string `json:"price"`
	IsActive    *bool   `json:"is_active"`
	IsFeatured  *bool   `json:"is_featured"`
	CategoryIDs []int64 `json:"category_ids"`
	// OptionGroups nil ise gruplar DEĞİŞMEZ; boş dizi hepsini kaldırır
	// (CategoryIDs ile aynı PATCH semantiği).
	OptionGroups []optionGroupLinkRequest `json:"option_groups"`
}

// list GET /api/admin/products — pasifler dahil
func (h *productHandler) list(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 24)
	offset := 0
	if page := c.QueryInt("page", 1); page > 1 {
		offset = (page - 1) * limit
	}

	list, err := h.svc.ListAdmin(c.Context(), limit, offset)
	if err != nil {
		return api.WriteError(c, err)
	}

	ids := make([]int64, 0, len(list))
	for _, p := range list {
		ids = append(ids, p.ID)
	}
	grouped, err := h.imgSvc.ListByProducts(c.Context(), ids)
	if err != nil {
		return api.WriteError(c, err)
	}

	return c.JSON(toProductViews(list, h.imgSvc, grouped))
}

// get GET /api/admin/products/:id — pasif olsa da döner
func (h *productHandler) get(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	p, err := h.svc.GetByID(c.Context(), int64(id))
	if err != nil {
		return api.WriteError(c, err)
	}

	imgs, err := h.imgSvc.ListByProduct(c.Context(), p.ID)
	if err != nil {
		return api.WriteError(c, err)
	}

	// Panelde pasif gruplar da görünmeli ki esnaf durumu anlasın.
	groups, err := h.optSvc.GroupsForProduct(c.Context(), p.ID, false)
	if err != nil {
		return api.WriteError(c, err)
	}

	view := toProductView(*p, h.imgSvc, imgs)
	view.OptionGroups = toProductOptionGroupViews(groups)
	return c.JSON(view)
}

// create POST /api/admin/products
func (h *productHandler) create(c *fiber.Ctx) error {
	var req createProductRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return badRequest(c, "Geçersiz fiyat")
	}

	// Gruplar ürün kaydedilmeden ÖNCE doğrulanır: geçersiz group_id veya
	// tekrar eden grup varsa ürün hiç oluşturulmaz. category_ids'in aksine
	// SetProductGroups ayrı bir çağrı olduğu için (productoption paketi
	// product paketine bağımlı, tersi import cycle yaratır — tek transaction
	// mümkün değil), doğrulamayı öne almak "ürün kaydedildi ama grup bağlama
	// başarısız" kısmi başarısını DB arızası dışındaki tüm durumlarda önler.
	links := toGroupLinks(req.OptionGroups)
	if req.OptionGroups != nil {
		if err := h.optSvc.ValidateGroupLinks(c.Context(), links); err != nil {
			return api.WriteError(c, err)
		}
	}

	in := product.CreateInput{
		Name:        req.Name,
		Description: req.Description,
		Price:       price,
		IsActive:    true, // varsayılan aktif (spec §3.2)
		CategoryIDs: req.CategoryIDs,
	}
	if req.IsActive != nil {
		in.IsActive = *req.IsActive
	}
	if req.IsFeatured != nil {
		in.IsFeatured = *req.IsFeatured
	}

	p, err := h.svc.Create(c.Context(), in)
	if err != nil {
		return api.WriteError(c, err)
	}

	// Gruplar ürün kaydedildikten sonra bağlanır — ürün id'si gerekiyor.
	// nil ise dokunulmaz (PATCH semantiği). Yukarıda doğrulandığı için
	// buradaki SetProductGroups'ta ErrInvalidInput pratikte beklenmez;
	// hata kontrolü DB arızası gibi ayrı bir sınıf için korunuyor.
	if req.OptionGroups != nil {
		if err := h.optSvc.SetProductGroups(c.Context(), p.ID, links); err != nil {
			return api.WriteError(c, err)
		}
	}

	groups, err := h.optSvc.GroupsForProduct(c.Context(), p.ID, false)
	if err != nil {
		return api.WriteError(c, err)
	}

	// Yeni ürünün henüz görseli yok.
	view := toProductView(*p, h.imgSvc, nil)
	view.OptionGroups = toProductOptionGroupViews(groups)
	return c.Status(fiber.StatusCreated).JSON(view)
}

// update PATCH /api/admin/products/:id
func (h *productHandler) update(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	var req updateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "Geçersiz istek")
	}

	// Gruplar ürün güncellenmeden ÖNCE doğrulanır — bkz. create handler'daki
	// aynı gerekçe: SetProductGroups ayrı bir çağrı, tek transaction'a
	// alınamıyor (import cycle), doğrulamayı öne almak kısmi başarıyı önler.
	links := toGroupLinks(req.OptionGroups)
	if req.OptionGroups != nil {
		if err := h.optSvc.ValidateGroupLinks(c.Context(), links); err != nil {
			return api.WriteError(c, err)
		}
	}

	in := product.UpdateInput{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
		IsFeatured:  req.IsFeatured,
		CategoryIDs: req.CategoryIDs,
	}

	if req.Price != nil {
		price, err := decimal.NewFromString(*req.Price)
		if err != nil {
			return badRequest(c, "Geçersiz fiyat")
		}
		in.Price = &price
	}

	p, err := h.svc.Update(c.Context(), int64(id), in)
	if err != nil {
		return api.WriteError(c, err)
	}

	// Gruplar ürün kaydedildikten sonra bağlanır — ürün id'si gerekiyor.
	// nil ise dokunulmaz (PATCH semantiği). Yukarıda doğrulandığı için
	// buradaki SetProductGroups'ta ErrInvalidInput pratikte beklenmez;
	// hata kontrolü DB arızası gibi ayrı bir sınıf için korunuyor.
	if req.OptionGroups != nil {
		if err := h.optSvc.SetProductGroups(c.Context(), p.ID, links); err != nil {
			return api.WriteError(c, err)
		}
	}

	imgs, err := h.imgSvc.ListByProduct(c.Context(), p.ID)
	if err != nil {
		return api.WriteError(c, err)
	}

	groups, err := h.optSvc.GroupsForProduct(c.Context(), p.ID, false)
	if err != nil {
		return api.WriteError(c, err)
	}

	view := toProductView(*p, h.imgSvc, imgs)
	view.OptionGroups = toProductOptionGroupViews(groups)
	return c.JSON(view)
}

// delete DELETE /api/admin/products/:id
func (h *productHandler) delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return badRequest(c, "Geçersiz id")
	}

	// Önce saklamadaki dosyaları temizle — ürün silinince product_images
	// CASCADE ile gider ve key'ler öğrenilemez hale gelir (spec §4.4).
	if err := h.imgSvc.DeleteAllForProduct(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}

	if err := h.svc.Delete(c.Context(), int64(id)); err != nil {
		return api.WriteError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
