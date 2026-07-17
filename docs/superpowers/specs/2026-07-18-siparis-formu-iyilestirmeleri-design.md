# Sipariş Formu İyileştirmeleri — Design

**Durum:** Onaylandı
**İlişkili:** `docs/superpowers/specs/2026-07-17-faz2-sepet-siparis-design.md` (Faz 2 — sepet ve sipariş)

## Amaç

Faz 2'de sipariş formu (`frontend/app/app/pages/siparis/index.vue`) çalışır durumda
ama dört noktada sürtünme/eksiklik var:

1. E-posta alanı zorunlu değil ama formda duruyor — gereksiz sürtünme.
2. Teslimat adresi tek serbest metin alanı; hangi ilçeye gönderim yapılabildiği
   belli değil (şu an sadece İzmir/Ödemiş + yakın ilçeler).
3. Tarih/saat seçimi native `<input type=date>` + `<select>` — hızlı seçim yok.
4. Kart mesajına gönderenin ismini eklemek için müşteri elle yazmak zorunda.

## Kapsam Dışı

- İlçeye göre farklı teslimat ücreti (spec'in "esnaftan öğrenilecek" notuyla
  aynı gerekçeyle Faz 2 kapsamı dışında bırakıldı).
- `districts` tablosu / admin'den ilçe yönetimi (YAGNI — ilçe sayısı az, nadiren
  değişir; config yeterli).
- Saat aralıklarının değiştirilmesi (kullanıcının paylaştığı ekran görüntüsü
  mockup'tı, mevcut 3 slot — 09:00-12:00 / 12:00-15:00 / 15:00-18:00 — kalıyor).

## 1. E-posta Alanının Kaldırılması

`buyer_email` backend'de zaten opsiyonel (`omitempty`). Sadece
`siparis/index.vue`'daki e-posta input'u ve `form.buyerEmail` alanı kaldırılır.
`createOrder` çağrısında `email` alanı gönderilmez. Backend/DB/migration
dokunulmaz — ileride ihtiyaç olursa (ör. admin elle giriyor) altyapı hazır kalır.

## 2. Teslimat İlçesi Seçimi

**Backend:**
- `pkg/config/config.go`: `Config.DeliveryDistricts []string` eklenir.
  `loadDelivery()` içine `DELIVERY_DISTRICTS` env'i eklenir — `DeliverySlots`
  ile birebir aynı desen (virgülle ayrılmış, trim'lenmiş, varsayım:
  `Ödemiş,Tire,Bayındır,Kiraz,Beydağ`).
- `internal/order/service.go`: `DeliveryConfig.Districts []string` eklenir.
  `CreateInput.District string` eklenir. `validateDelivery` içinde slot
  kontrolüyle aynı desende `slices.Contains(s.cfg.Districts, in.District)`
  doğrulaması eklenir — listede yoksa `errorsx.ErrInvalidInput`.
- `internal/order/model.go`: `Order.District string` ve `NewOrder.District string`
  eklenir.
- **Migration (000006):** `orders` tablosuna `delivery_district TEXT NOT NULL`
  kolonu eklenir (up/down).
- `internal/api/app/order_view.go`: `createOrderRequest.Delivery.District` eklenir;
  `deliveryConfigResponse.Districts []string` eklenir.
- `internal/api/app/order_handler.go`: `req.Delivery.District` → `CreateInput.District`
  aktarımı, `deliveryConfig` handler'ına `Districts` eklenir.
- `internal/api/idare/order_view.go`: `orderView.DeliveryDistrict string` eklenir
  (admin adres bilgisinin yanında ilçeyi görsün).
- **Ücrete etkisi yok** — `DeliveryFee` hesaplaması değişmez, ilçe sadece bilgi.

**Frontend (app):**
- `types/api.ts`: `DeliveryConfig.districts: string[]`, `CreateOrderInput.delivery.district: string` eklenir.
- `siparis/index.vue`: "Teslimat" fieldset'ine ilçe `<select>` eklenir
  (`cfg.value?.districts` ile doldurulur), zorunlu alan.

**Frontend (idare):**
- `model/order.ts`: `Order.delivery_district: string` eklenir.
- `siparisler/[id].vue`: Teslimat kartında adres satırının yanına ilçe eklenir.

## 3. Hazır Tarih/Saat Çipleri

Yalnızca `siparis/index.vue` değişir — backend/config dokunulmaz.

**Tarih:** Bugünden başlayarak ilk 3 gün çip olarak gösterilir:
- 1. çip: "Bugün" + tarih (gün.ay)
- 2. çip: "Yarın" + tarih
- 3. çip: haftanın günü (ör. "Pazar") + tarih
- 4. çip: takvim ikonu, "Takvim" etiketi — tıklanınca gizli
  `<input type=date>` programatik olarak açılır (`showPicker()` veya native
  click tetiklemesi), `min`/`max` mevcut mantıkla aynı (`bugun` / `sonTarih`).
  Seçilen tarih 4 gün ve sonrası ise 4. çip seçili tarihi gösterir.

Seçili çip vurgulanır (mevcut `btn-primary` rengiyle tutarlı, DESIGN.md'deki
`bg-primary` tonuyla). Seçim `form.date`'e ISO string (`YYYY-MM-DD`) olarak yazılır.

**Saat:** Mevcut `cfg.slots` listesi `<select>` yerine çip grubu olarak
gösterilir, seçili slot vurgulanır. Aynı gün + cutoff sonrası kısıtı
backend zaten doğruluyor (`service.go` `pastCutoff`); frontend bu çipleri
disable etmez (mevcut davranış — backend hatası formda Türkçe mesajla gösterilir).

## 4. Kart Mesajına İsim Ekleme

Kart mesajı `<textarea>`'sının altına checkbox eklenir: "İsmim eklensin".
Backend'e ayrı alan **eklenmez** — `gonder()` fonksiyonunda gönderim anında
`card_message` sonuna otomatik eklenir:

```
{form.cardMessage}

- {form.buyerName}
```

Checkbox işaretli değilse mesaj olduğu gibi gönderilir. `card_message` boşsa
ve checkbox işaretliyse sadece isim gönderilir (boş satır olmadan).

## Test Planı

- Backend: `service_test.go`'ya ilçe doğrulama testleri (`TestService_Create_GecersizIlceReddedilir`
  — mevcut `GecersizSlotReddedilir` deseniyle aynı). `store_test.go`'daki
  `testNewOrder()` helper'ına `District` eklenir. Config testine
  `TestLoad_DeliveryDistrictsFromEnv` eklenir.
- Frontend: Çip bileşenleri için ayrı bir vitest gerekmiyor (saf UI, mevcut
  `cartLogic.ts` gibi test edilebilir saf fonksiyon yok) — mevcut build ve
  manuel doğrulama yeterli, plan'daki Task 9 deseniyle tutarlı.

## Geriye Dönük Uyumluluk

`delivery_district` `NOT NULL` yeni kolon — mevcut (varsa) siparişler için
migration'da `DEFAULT ''` YOK, çünkü Faz 2 henüz prod'a çıkmadı (dev/test
DB'lerinde birkaç test siparişi var, önemli değil). Migration doğrudan
`NOT NULL` ile eklenir; gerekirse `make migrate-down` ile Faz 2'nin son
migration'ına dönülebilir.
