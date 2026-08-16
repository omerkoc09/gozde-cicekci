# Ürün Özelleştirme ("Buket Tasarla") — Tasarım

Tarih: 2026-08-16
Durum: onaylandı, uygulanmayı bekliyor

## 1. Amaç

Müşteri ürünü sipariş ederken görünüşünü seçebilsin: ambalaj rengi, kurdele
rengi, kutu rengi gibi. Bu **genel bir özellik** — belirli bir ürün tipine
gömülü değil, her ürüne açılabilir.

Esnaf seçenek gruplarını ve renkleri panelden kendisi tanımlar; ileride
"Çiçek Rengi" veya "Vazo Tipi" gerekirse kod değişikliği ya da migration
olmadan ekleyebilir.

## 2. Kararlar

Brainstorming'de verilen kararlar, gerekçeleriyle:

| Karar | Seçilen | Gerekçe |
|---|---|---|
| Seçenek tanımı | Merkezi havuz + ürün başına aç/kapa | Renk bir kez tanımlanır, tüm ürünlerde kullanılır. Esnaf her üründe aynı renkleri tekrar yazmaz. |
| Değer tipi | Grup başına `color` \| `text` | "Buket Boyu: Küçük/Orta/Büyük" gibi renksiz seçenekler de sonradan eklenebilsin, migration gerekmesin. |
| Fiyat etkisi | Yok | MVP sadeliği. Fiyat farkı gerekirse `option_values`'a bir kolon eklenir; sepet/PayTR tutar doğrulaması bugün hiç etkilenmiyor. |
| Zorunluluk | Ürün başına `is_required` | Bazı üründe renk kritik, bazısında "çiçekçi seçsin" makul. Kararı esnaf verir. |
| Sepet davranışı | Farklı seçim = ayrı kalem | Pembe ambalajlı buket ile beyaz ambalajlı aynı buket farklı hazırlanır. Tek kalemde birleşirse müşteri iki farklı renk sipariş edemez. |
| Panelde görünüm | Renk noktası + ad | Esnaf bir bakışta görsün, yazıcı çıktısında da okunur olsun. |

## 3. Veri modeli (migration 10)

```sql
option_groups
  id          BIGSERIAL PK
  name        TEXT NOT NULL              -- "Ambalaj Rengi"
  slug        TEXT NOT NULL UNIQUE       -- "ambalaj-rengi"
  kind        TEXT NOT NULL CHECK (kind IN ('color','text'))
  sort_order  INT  NOT NULL DEFAULT 0
  is_active   BOOLEAN NOT NULL DEFAULT true

option_values
  id          BIGSERIAL PK
  group_id    BIGINT NOT NULL REFERENCES option_groups(id) ON DELETE CASCADE
  name        TEXT NOT NULL              -- "Pembe"
  swatch_hex  TEXT NOT NULL DEFAULT ''   -- "#F5A9C8"; kind='text' ise boş
  sort_order  INT  NOT NULL DEFAULT 0
  is_active   BOOLEAN NOT NULL DEFAULT true

product_option_groups                    -- hangi üründe hangi grup açık
  product_id  BIGINT NOT NULL REFERENCES products(id)      ON DELETE CASCADE
  group_id    BIGINT NOT NULL REFERENCES option_groups(id) ON DELETE CASCADE
  is_required BOOLEAN NOT NULL DEFAULT false
  PRIMARY KEY (product_id, group_id)

order_item_options                       -- sipariş anındaki seçim (KOPYA)
  id            BIGSERIAL PK
  order_item_id BIGINT NOT NULL REFERENCES order_items(id) ON DELETE CASCADE
  group_name    TEXT NOT NULL            -- "Ambalaj Rengi"
  value_name    TEXT NOT NULL            -- "Pembe"
  swatch_hex    TEXT NOT NULL DEFAULT ''
  sort_order    INT  NOT NULL DEFAULT 0  -- gruptaki sıra, gösterimde korunur
```

**Neden kopya:** `order_item_options` gruba/değere referans tutmaz, isim ve
hex'i kopyalar. Esnaf sonradan "Pembe"yi silerse veya rengini değiştirirse
eski siparişin ne olduğu bilgisi bozulmamalı. Bu, `order_items.product_name`
ve `price_at_order`'daki mevcut desenin aynısı.

**Ürün başına sıra yok:** Gruplar `option_groups.sort_order` ile sıralanır;
ürün bazında ayrı sıra tutulmuyor (YAGNI). Gerekirse
`product_option_groups`'a kolon eklenir.

**Seed:** Migration üç grubu `kind='color'` olarak ekler — Ambalaj Rengi,
Kurdele Rengi, Kutu Rengi ve örnek fotoğraftaki renkler. Bunlar başlangıç
verisi; panelden düzenlenebilir/silinebilir, kodda gömülü değil.

**Geri alma:** `000010_product_options.down.sql` dört tabloyu düşürür.
`order_item_options` da düşeceği için geçmiş siparişlerin renk seçimleri
kalıcı olarak kaybolur — down yalnızca geliştirme içindir. Prod'da
migration compose'un `migrate` servisiyle ileri yönlü uygulanır.

## 4. Backend

Yeni paket: `internal/productoption/` — `model.go`, `store.go`, `service.go`.
Mevcut `category` paketinin desenini takip eder.

### 4.1 Admin uçları (`/api/admin`)

```
GET    /option-groups                    gruplar + değerleri (pasifler dahil)
POST   /option-groups                    {name, kind}
PATCH  /option-groups/:id                {name?, is_active?}
PUT    /option-groups/reorder            {ids}
DELETE /option-groups/:id
GET    /option-groups/:id/product-count  silme öncesi uyarı için

POST   /option-groups/:id/values         {name, swatch_hex}
PATCH  /option-values/:id                {name?, swatch_hex?, is_active?}
PUT    /option-groups/:id/values/reorder {ids}
DELETE /option-values/:id
```

Sıralama madde 2'deki desenin aynısı: liste tamamını ister, tek transaction,
eksik/tekrarlı/yabancı liste 400. `kind` oluşturduktan sonra değişmez —
`color`'dan `text`'e geçiş mevcut hex'leri anlamsız kılar (kategorideki
`axis` kuralıyla aynı gerekçe).

Ürün formu için mevcut ürün uçları genişler:
`POST/PATCH /admin/products` gövdesine `option_groups: [{group_id, is_required}]`
eklenir. `nil` ise değişmez, boş dizi ise hepsi kaldırılır — `CategoryIDs`'in
PATCH semantiğiyle aynı.

### 4.2 Public uçları

`GET /api/products/:slug` yanıtına eklenir:

```json
"option_groups": [
  { "id": 1, "name": "Ambalaj Rengi", "kind": "color", "is_required": true,
    "values": [ { "id": 5, "name": "Pembe", "swatch_hex": "#F5A9C8" } ] }
]
```

Yalnızca **aktif** grup ve **aktif** değerler döner. Pasife alınan bir renk
yeni siparişlerde görünmez, eski siparişleri etkilemez.

### 4.3 Sipariş oluşturma — güvenlik

`CreateInput` yorumundaki ilke aynen uygulanır: *"FİYAT YOK — sunucu DB'den
okur."* Seçimler için de tarayıcı yalnızca **id** gönderir:

```go
type CreateItem struct {
    ProductID int64
    Quantity  int
    OptionValueIDs []int64   // YENİ — yalnızca id
}
```

Sunucu `order.Service.Create` içinde, ürün okunduktan sonra doğrular:

1. Her `value_id` gerçekten var mı ve **aktif** mi
2. Değerin grubu **bu ürüne açık** mı (`product_option_groups`)
3. Aynı gruptan **birden fazla** değer gönderilmiş mi → reddet
4. Ürünün **zorunlu** grupları eksiksiz doldurulmuş mu

Geçmezse `ErrInvalidInput` ile sipariş reddedilir. `group_name`, `value_name`
ve `swatch_hex` **DB'den okunur**, tarayıcıdan gelen isimlere güvenilmez.

Yeni bir dar arayüz gerekiyor (`ProductReader` desenine paralel):

```go
// OptionReader order paketinin seçenek doğrulaması için ihtiyaç duyduğu tek şey.
type OptionReader interface {
    ResolveForProduct(ctx context.Context, productID int64, valueIDs []int64) ([]ResolvedOption, error)
    RequiredGroupIDs(ctx context.Context, productID int64) ([]int64, error)
}
```

Böylece `order` paketi `productoption` paketinin tamamına bağlanmaz.

**Ödeme akışına etki yok.** Fiyat farkı olmadığı için `itemsTotal`, PayTR
sepeti ve callback'teki tutar doğrulaması ([service.go:308](../../../backend/internal/order/service.go#L308))
aynen çalışır.

**Sipariş görünümü:** `OrderItem`'a `Options []OrderItemOption` eklenir.
`Store.List`'teki mevcut N+1 dersine uyulur — seçimler `itemsOfMany` gibi
tek batch sorguyla (`WHERE order_item_id = ANY($1)`) çekilir, kalem başına
ayrı sorgu açılmaz.

## 5. Admin paneli

### 5.1 Yeni sayfa `/idare/secenekler`

Kategoriler sayfasının deseninde, iki sütun:

- **Sol — gruplar:** ad, tip rozeti (Renk/Metin), değer sayısı, aktif switch,
  ▲▼ sıralama, düzenle/sil
- **Sağ — seçili grubun değerleri:** ad + renk kutusu (`<input type="color">`
  ve hex girişi yan yana), aktif switch, ▲▼ sıralama, sil

`kind='text'` grupta renk kutusu gösterilmez.

Silme öncesi uyarı, kategori silmedeki mevcut desen:
> "Ambalaj Rengi 12 üründe kullanılıyor. Silerseniz bu ürünlerde artık
> sorulmayacak (geçmiş siparişler etkilenmez). Devam edilsin mi?"

Navigasyona "Seçenek Yönetimi" eklenir.

### 5.2 Ürün formuna "Özelleştirme" bölümü

Her aktif grup için bir satır:

```
[✓] Ambalaj Rengi   [ ] Zorunlu     ● ● ● ● ● ● (grubun renkleri, salt okunur)
[ ] Kurdele Rengi   [ ] Zorunlu
```

İşaretlenmemiş grup o üründe hiç sorulmaz. "Zorunlu" yalnızca grup
işaretliyken etkin.

### 5.3 Sipariş detayı

Kalem adının altında seçimler:

```
Kırmızı Gül Buketi × 1
  ● Ambalaj: Pembe   ● Kurdele: Beyaz
```

Renk noktası `swatch_hex` ile boyanır; hex boşsa (`kind='text'`) yalnızca
metin. Yazıcı çıktısında da okunur.

## 6. Public site

### 6.1 Ürün detay sayfası

Galerinin altında, "Sepete Ekle" butonunun üstünde:

```
Ambalaj Rengi *     ◉ ○ ○ ○ ○ ○      ← kind='color': yuvarlak renk noktaları
Buket Boyu          [Küçük] [Orta]   ← kind='text': seçilebilir etiketler
```

- Seçili değer halkayla belirtilir (örnek fotoğraftaki gibi)
- Zorunlu grup seçilmeden "Sepete Ekle" pasif; altında "Ambalaj rengi seçiniz"
- Erişilebilirlik: renk noktaları `role="radiogroup"` içinde, her biri
  `aria-label` ile renk adını taşır — renk tek başına bilgi taşımamalı

### 6.2 Sepet — birleştirme anahtarı değişiyor

`cartLogic.ts`'teki üç fonksiyon da bugün `product_id`'yi anahtar kabul
ediyor. Üçü birden değişmeli:

| Fonksiyon | Bugün | Sonra |
|---|---|---|
| `addItem` | `i.product_id === yeni.product_id` | ürün + seçim kümesi eşitse birleştir |
| `removeItem` | `product_id` ile filtrele | satır anahtarı ile filtrele |
| `setItemQuantity` | `product_id` ile bul | satır anahtarı ile bul |

Satır anahtarı: `product_id` + sıralanmış `option_value_ids`
(ör. `"12:3,7"`). Sıralama şart — `[7,3]` ile `[3,7]` aynı satır olmalı.

`CartItem`'a `options: {group_name, value_name, swatch_hex, value_id}[]`
eklenir; gösterim için isim/hex, gönderim için id. Sepet çekmecesinde ve
sipariş özetinde küçük renk noktaları görünür.

**Geriye dönük uyum:** Sepet `localStorage`'da yaşıyor. Bu değişiklikten
önce sepet kurmuş bir müşterinin kalemlerinde `options` alanı yok —
`undefined`'ı boş dizi kabul eden okuma bunu sorunsuz karşılar, sepet
sıfırlanmaz.

### 6.3 Sipariş gönderimi

`useOrders` gövdesine kalem başına `option_value_ids` eklenir. İsim/hex
gönderilmez — sunucu DB'den okur (§4.3).

## 7. Test planı

TDD; `make test` (`-p 1` — `go test ./...` kullanılmıyor, DURUM.md).

**Backend birim:**
- Grup/değer CRUD, `kind` değişmezliği, sıralama (madde 2 desenindeki 4 vaka)
- Pasif grup/değer public yanıtta yok
- Sipariş doğrulaması: yabancı değer, pasif değer, ürüne kapalı grup,
  aynı gruptan iki değer, eksik zorunlu grup → hepsi `ErrInvalidInput`
- Seçimler `order_item_options`'a **kopyalanıyor**; değer sonradan silinince
  eski sipariş bozulmuyor
- Kalem seçimleri tek batch sorguyla okunuyor (N+1 regresyon testi)

**Frontend birim:** `cartLogic.test.ts` genişler — aynı ürün farklı seçimle
ayrı kalem, aynı seçim (farklı sırayla verilmiş) tek kalem, seçimsiz eski
kalemler bozulmuyor.

**E2E — proxy üzerinden.** Cookie ve sipariş akışları Nuxt proxy'si üzerinden
doğrulanır, Go'ya doğrudan curl atılmaz: doğrudan çağrı proxy katmanındaki
hataları gizler (bu projede daha önce yaşandı).

**Tarayıcıda:** panelde grup/renk ekleme → üründe aktifleştirme → müşteri
sayfasında seçim → sepette ayrı kalem → sipariş → panelde renk noktalarıyla
görünüm. Bu zincirin tamamı gerçek tarayıcıda çalışmadan iş bitmiş sayılmaz —
"görsel bölümü kaydettikten sonra açılmıyordu" hatası yalnızca tarayıcıda
görünmüştü, curl ile görünmezdi.

## 8. Kapsam dışı

- Fiyat farkı (`price_delta`) — gerekirse tek kolon + hesap noktası
- Seçeneğe göre stok takibi
- Seçenek başına görsel (renk seçilince ürün fotoğrafının değişmesi)
- Ürün bazında değer alt kümesi ("bu bukette sadece 4 renk") — brainstorming'de
  değerlendirildi, merkezi havuz yeterli bulundu
- Sürükle-bırak sıralama — ok butonları yeterli
