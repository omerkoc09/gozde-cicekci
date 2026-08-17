# Stok Yönetimi ve İndirimli Ürünler — Tasarım

Tarih: 2026-08-17
Durum: onaylandı, uygulanmayı bekliyor

## 1. Amaç

İki ayrı ama aynı veri modeline dokunan özellik:

**Stok yönetimi.** Esnaf istediği ürüne stok adedi tanımlayabilsin. Siteden
satış olduğunda stok otomatik azalsın; WhatsApp üzerinden yapılan satışlar
takip edilemediği için panelden elle de düşürülebilsin. Stok bittiğinde ürün
sepete eklenemesin ve müşteriye "Tükendi" bilgisi verilsin.

**İndirimli Ürünler.** Esnaf bazı ürünleri geçici olarak indirime soksun,
bunlar ayrı bir sayfada toplansın. Müşteri hem eski hem yeni fiyatı görsün.
Her indirimin bir adet kotası olsun; o sayıya ulaşıldığında indirim
kendiliğinden kalksın.

Stok takibi **ürün başına isteğe bağlı** — bugünkü 40 ürün hiçbir işlem
yapılmadan aynen çalışmaya devam eder.

## 2. Kararlar

Brainstorming'de verilen kararlar, gerekçeleriyle:

| Karar | Seçilen | Gerekçe |
|---|---|---|
| Stok düşme anı | Rezervasyon + callback'te kesinleşme | Sipariş ödemeden ÖNCE oluşuyor (`awaiting_payment`). Sadece callback'te düşülse son ürünü iki müşteri aynı anda ödeyebilir; sipariş anında düşülse ödemeden vazgeçen müşteri stoğu kilitler. |
| Ödenmemiş rezervasyon | 20 dk sonra otomatik serbest | Yarım kalan ödeme stoğu sonsuza kadar tutmamalı. PayTR oturumu için 20 dk fazlasıyla yeterli. |
| Manuel düşme | Ürün listesinde `+`/`−` + sebep | Esnaf telefondayken saniyeler içinde yapabilmeli. Sebep kaydı "bu ay WhatsApp'tan kaç sattım" sorusunu cevaplar. |
| Takipsiz ürün | Sınırsız satılır, hiç tükenmez | `track_stock=false` varsayılan → mevcut ürünler ve site davranışı hiç değişmez. Migration kimseyi tükendi göstermez. |
| Tükendi davranışı | Görünür kalır, rozet + pasif buton | Ürün gizlenirse müşteri linkte 404 alır, esnaf da ürünün kaybolduğunu fark etmez. Görünür kalırsa WhatsApp'tan sorulabilir — satış fırsatı kaybolmaz. |
| Sepette stok azalması | Ödemeye geçerken sunucuda kontrol | Sepet localStorage'da; tarayıcıdaki veriye güvenilmez. Fiyat kuralının aynısı: son sözü sunucu söyler. |
| İndirim kategorisi | Otomatik/sanal kategori | `categories.axis` yalnızca `occasion`/`type` ve değiştirilemez. İndirim ikisine de uymuyor, üyeliği de türetilmiş (kota bitince çıkmalı). Gerçek kategori olsaydı esnaf iki yeri elle senkron tutardı. |
| Kota sayımı | Site + WhatsApp satışları | Esnafın gerçek indirim maliyetini yansıtır. Elle düşürmede "indirimli satış" işaretlenir. |
| Kota bitince | İndirim kalkar, ürün normal fiyattan satılır | Stok ve indirim bağımsız iki kavram. Stok varsa satış sürer. |

## 3. Veri modeli (migration 11)

### 3.1 `products` tablosuna eklenen kolonlar

```sql
ALTER TABLE products
    ADD COLUMN track_stock     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN stock_quantity  INT     NOT NULL DEFAULT 0,
    ADD COLUMN stock_reserved  INT     NOT NULL DEFAULT 0,
    ADD COLUMN discount_price  NUMERIC(10,2),
    ADD COLUMN discount_quota  INT,
    ADD COLUMN discount_sold   INT     NOT NULL DEFAULT 0;

ALTER TABLE products
    ADD CONSTRAINT products_stock_nonneg
        CHECK (stock_quantity >= 0 AND stock_reserved >= 0),
    ADD CONSTRAINT products_discount_sold_nonneg
        CHECK (discount_sold >= 0),
    -- discount_price varsa quota da olmalı: kotasız indirim süresiz indirimdir,
    -- bu özelliğin amacı değil.
    ADD CONSTRAINT products_discount_pair
        CHECK ((discount_price IS NULL AND discount_quota IS NULL)
            OR (discount_price IS NOT NULL AND discount_quota > 0));

-- İndirimli ürünler sayfası bu koşulla okuyor
CREATE INDEX idx_products_discount_active ON products (id)
    WHERE discount_price IS NOT NULL;
```

`DEFAULT false` kritik: mevcut ürünler takipsiz başlar, site aynen çalışır.

`stock_reserved` **ödeme bekleyen** adet. Fiziksel stok `stock_quantity`,
müşteriye satılabilir olan `stock_quantity - stock_reserved`.

### 3.2 `stock_movements` — hareket kaydı

```sql
CREATE TABLE stock_movements (
    id             BIGSERIAL PRIMARY KEY,
    product_id     BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    delta          INT NOT NULL,          -- negatif: düşüş, pozitif: giriş
    reason         TEXT NOT NULL CHECK (reason IN
                     ('siparis','whatsapp_satisi','sayim_duzeltme',
                      'yeni_parti','iptal_iade','rezervasyon_iptal')),
    -- Sipariş silinirse hareket kaydı ölmemeli (order_items.product_id deseni)
    order_id       BIGINT REFERENCES orders(id) ON DELETE SET NULL,
    -- İndirim kotasının WhatsApp satışlarını da sayabilmesi için
    was_discounted BOOLEAN NOT NULL DEFAULT false,
    note           TEXT NOT NULL DEFAULT '',
    -- Hareketi yapan panel kullanıcısı; kullanıcı silinse de kayıt kalır
    admin_user_id  BIGINT REFERENCES admin_users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_stock_movements_product
    ON stock_movements (product_id, created_at DESC);
```

Bu tablo iki soruyu cevaplar: "bu ay WhatsApp'tan kaç sattım" ve "stok neden
bu sayıda". Stok bir gün tutarsız görünürse neyin ne zaman olduğu buradan
okunur.

### 3.3 Türetilen kavramlar

Kodda **tek yerde** tanımlanır, her yerde aynı hesap kullanılır:

```
satılabilir(p) = p.track_stock ? p.stock_quantity - p.stock_reserved : ∞
indirimAktif(p) = p.discount_price IS NOT NULL
                  AND p.discount_sold < p.discount_quota
geçerliFiyat(p) = indirimAktif(p) ? p.discount_price : p.price
```

Kota dolduğu an sorgu indirimi görmez — ayrıca "indirimi kapat" işi çalıştırmaya
gerek yok, kendiliğinden söner.

## 4. Stok akışı

### 4.1 Rezervasyon — yarış durumu koruması

Sipariş oluşurken, **tek SQL ifadesiyle**:

```sql
UPDATE products
   SET stock_reserved = stock_reserved + $qty
 WHERE id = $id
   AND (NOT track_stock OR stock_quantity - stock_reserved >= $qty)
RETURNING id;
```

Satır dönmezse stok yetmemiştir → sipariş reddedilir.

**"Önce oku, sonra karar ver, sonra yaz" YAPILMAZ** — okuma ile yazma arasında
başka istek araya girebilir. Koşul `WHERE` içinde olduğu için Postgres satırı
kilitler ve eşzamanlı iki istekten yalnızca biri kazanır.

`NOT track_stock` kısayolu: takipsiz ürünler bu kontrole hiç takılmaz.

Çağrı `order.Service.Create()` içinde, **sipariş kaydı ile aynı transaction'da**
yapılır. Ürünler `product_id` sırasına dizilerek rezerve edilir — iki müşteri
aynı iki ürünü ters sırayla sepete koyduğunda deadlock olmasın diye.

### 4.2 Rezervasyonun üç çıkışı

| Olay | İşlem | Hareket kaydı |
|---|---|---|
| Ödeme başarılı (`callback_ok`) | `reserved -= q`, `quantity -= q` | `siparis` |
| Ödeme gelmez (20 dk) | `reserved -= q` | `rezervasyon_iptal` |
| İade (`refund`) | `quantity += q` | `iptal_iade` |

Kesinleşme, siparişi `paid` yapan **aynı transaction'a** girer. `ApplyCallback`
zaten `callback_ok` olayıyla idempotent — PayTR aynı callback'i iki kez
gönderirse stok iki kez düşmez. Aynı koruma bedavaya devralınır.

### 4.3 Süpürücü (sweeper)

5 dakikada bir çalışan goroutine: `awaiting_payment` durumundaki ve 20
dakikadan eski siparişlerin rezervasyonlarını serbest bırakır.

- Sunucu başlatılırken `main.go`'da başlar, `context` ile kapanır.
- Serbest bırakılan her rezervasyon `stock_movements`'a yazılır — sessizce
  kaybolan bir şey olmaz.
- Bir siparişte hata olursa loglanır ve diğerlerine devam edilir; tek hata tüm
  süpürmeyi durdurmamalı.

### 4.4 İndirim kotasının tüketimi

`discount_sold` **yalnızca ödeme kesinleştiğinde** artar (rezervasyonda
değil) — stok kesinleşmesiyle aynı transaction'da.

Fiyat, sipariş oluşturulurken sunucuda okunur ve `order_items.price_at_order`'a
yazılır. Müşteri indirimli fiyatı görüp, kota o sırada dolarsa bile **gördüğü
fiyattan öder** — sipariş anındaki fiyat bağlayıcıdır (mevcut `price_at_order`
deseninin aynısı; esnaf indirimi kaldırsa dünkü siparişin tutarı bozulmaz).

Manuel düşmede `was_discounted=true` işaretlenirse `discount_sold` de artar.

## 5. Panel (idare)

### 5.1 Ürün listesi — hızlı stok düşme

Her satırda stok sütunu ve `[−] 12 [+]` düğmeleri. `−` basıldığında açılır menü
sebep sorar:

```
 Gül Buketi      [ − ] 12 [ + ]   ⌄
   └ sebep: WhatsApp satışı ▼
     [✓] indirimli satış      ← yalnızca indirim aktifse görünür
     [Onayla]
```

- Varsayılan sebep **WhatsApp satışı** (en sık kullanılan).
- Ürün indirimliyse "indirimli satış" kutusu işaretli gelir.
- Takipsiz üründe sütun `—` gösterir, düğme yok.

### 5.2 Ürün düzenleme sayfası

```
Stok
  [✓] Stok takibi yap
      Stok adedi:  [ 12 ]        Rezerve: 2 (ödeme bekliyor)

İndirim
  Normal fiyat:      1.850 TL
  İndirimli fiyat:   [ 1.450 ]  TL
  Kaç adet indirimli:[ 10 ]      Satılan: 3 · Kalan: 7
  [İndirimi kaldır]
```

Ek olarak **Hareketler** sekmesi: o ürünün tarih/adet/sebep dökümü. Mevcut
"kullanan ürünler" sekmesi deseniyle aynı — yeni bir yapı icat edilmiyor.

Stok takibi kapatılırsa `stock_quantity` korunur (tekrar açılırsa eski değer
gelir), ama satılabilirlik sınırsıza döner.

**"İndirimi kaldır"** → `discount_price`, `discount_quota` NULL yapılır **ve
`discount_sold` sıfırlanır**. Sıfırlanmazsa esnaf aynı ürüne ikinci kez indirim
girdiğinde kota daha baştan dolmuş görünür. Geçmiş satışlar `stock_movements`'ta
zaten duruyor — `discount_sold` yalnızca **yürürlükteki** indirimin sayacı,
kalıcı bir toplam değil. Aynı sebeple yeni indirim girildiğinde de sıfırlanır.

## 6. Site (public)

### 6.1 Ürün kartı

İki bağımsız rozet, çakışabilirler:

```
┌──────────────┐        ┌──────────────┐
│   [görsel]   │        │   [görsel]   │
│  %22 İNDİRİM │        │   TÜKENDİ    │
│ Gül Buketi   │        │ Gül Buketi   │
│ 1̶.8̶5̶0̶ 1.450 TL│       │ 1.850 TL     │
│ [Sepete Ekle]│        │ [Sepete Ekle]│ ← pasif
└──────────────┘        │ [WhatsApp'tan sor] │
                        └──────────────┘
```

- Eski fiyat üstü çizili, indirimli fiyat vurgulu — **ikisi de görünür**.
- Tükenen ürün listede kalır, sepete ekle pasifleşir, yerine WhatsApp düğmesi
  çıkar. Mevcut `buildOrderMessage` yardımcısına "bu ürün tükendi, ne zaman
  gelir?" varyantı eklenir.
- İndirim yüzdesi eski/yeni fiyattan hesaplanır, ayrı alan tutulmaz.

### 6.2 İndirimli Ürünler sayfası

`/indirimli` — türetilmiş liste:

```sql
WHERE is_active
  AND discount_price IS NOT NULL
  AND discount_sold < discount_quota
```

`categories` tablosuna kayıt eklenmez, `axis` modeline dokunulmaz. Menüye tek
link girer. Esnaf indirimi girdiği an sayfa dolar, kota bitince ürün
kendiliğinden çıkar — senkron tutulacak ikinci bir yer yok.

Uygulama olarak `product.Filter`'a `DiscountedOnly bool` alanı eklenir ve
`ListPublic` içinde `FeaturedOnly` ile aynı desende sabit koşul olarak
uygulanır — mevcut filtre zinciri bozulmaz, sıralama (`created_at DESC`)
korunur.

### 6.3 Sepet

Sepet localStorage'da kalmaya devam eder. Stok kontrolü **ödemeye geçerken
sunucuda** yapılır; yetersizse hangi ürün için kaç adet kaldığı döner:

```
⚠ Gül Buketi için yalnızca 1 adet kaldı.
  [Adedi 1 yap]  [Sepetten çıkar]
```

Müşteri düzeltince ödeme devam eder.

## 7. API değişiklikleri

`Product` yanıtına eklenen alanlar:

| Alan | Tip | Açıklama |
|---|---|---|
| `old_price` | string\|null | İndirim aktifse normal fiyat, değilse null |
| `in_stock` | bool | Takipsizse her zaman true |
| `stock_quantity` | int\|null | Takipsizse null |
| `discount_remaining` | int\|null | Kalan indirimli adet |

Panel yanıtı ayrıca `stock_reserved` ve `track_stock` içerir.

Yeni uçlar:

```
GET   /api/products?discounted=true      indirimli ürünler
POST  /api/admin/products/:id/stock      manuel stok hareketi
GET   /api/admin/products/:id/movements  hareket geçmişi
```

`price` alanı **geçerli fiyatı** döner (indirim aktifse indirimli). Böylece
sepet ve toplam hesapları değişmeden çalışır.

## 8. Hata yönetimi

Yeni hata tipi eklenmiyor; `errorsx.ErrInvalidInput` yeterli. Stok yetmediğinde
mesaj hangi ürün ve kaç adet kaldığını söyler — müşterinin sepetini
düzeltebilmesi için şart.

Üç durum bilinçli olarak sessiz geçer:

- **Süpürücü bir rezervasyonu bırakamazsa** → logla, devam et. Bir sonraki tur
  tekrar dener.
- **Callback'te stok düşüşü başarısızsa** → ödeme de yazılmaz (aynı
  transaction), PayTR retry gönderir. Para hareketi ile stok asla ayrışamaz.
- **İade sonrası stok iadesi başarısızsa** → iade geçerli sayılır, KRİTİK
  loglanır. Para geri gitmişse geri alınamaz; stok esnaf tarafından düzeltilir.
  (Mevcut `callback_ok` yazımındaki gerekçenin aynısı.)

## 9. Test stratejisi

Projedeki servis/store ayrımı ve `_test.go` deseni izlenir.

| Test | Neyi kanıtlar |
|---|---|
| **Yarış durumu** — aynı ürüne eşzamanlı iki rezervasyon (gerçek DB, goroutine) | Tam olarak biri kazanır, `stock_reserved` stoğu aşmaz. Tasarımın en kritik iddiası. |
| **Idempotency** — aynı callback iki kez | Stok bir kez düşer |
| **Takipsiz ürün** — 1000 adet sipariş | Hiçbir stok alanı değişmez; mevcut ürünler bozulmaz |
| **Kota sınırı** — son adette iki eşzamanlı sipariş | Biri indirimli, diğeri normal fiyattan öder |
| **Süpürücü** — 20 dk geçmiş / 19 dk'lık rezervasyon | İlki serbest kalır, ikincisi kalmaz |
| **İade** | Stok geri eklenir, hareket kaydı yazılır |
| **Saf fonksiyonlar** — `indirimAktif`, `satılabilir` | Nuxt'tan bağımsız, `cartLogic.ts` gibi |

E2E tarafında sepet→ödeme akışı **Nuxt proxy üzerinden** doğrulanır — Go'ya
doğrudan curl atmak proxy hatalarını gizler.

## 10. Kapsam dışı

Bilinçli olarak yapılmıyor:

- **Stok uyarı eşiği / bildirim** ("3 adet kaldı" maili) — bildirim altyapısı
  yok, üyelikte de kapsam dışı bırakılmıştı.
- **Varyant bazlı stok** — renk seçenekleri stoğu paylaşır. Ambalaj rengi başına
  ayrı stok, `product_option_groups` modelini baştan yazmayı gerektirir.
- **Tarih aralıklı otomatik indirim** ("15–20 Ağustos arası") — kota ile aynı
  şeyi çözmüyor: kota indirimin **maliyetini** sınırlar, tarih **süresini**.
  İkisi birbirinin yerine geçmez ve esnaf ileride ikisini birden isteyebilir.
  Bu turda kapsam dışı çünkü (a) istenen özellik kota bazlıydı, (b) tarih
  desteği saat dilimi (Europe/Istanbul), "yarın başlıyor" durumu ve ödeme
  sırasında sınırı geçen müşteri gibi ek kararlar getiriyor.

  **Sonradan eklemek ucuz — şema bunu engellemiyor:**

  ```sql
  ALTER TABLE products
      ADD COLUMN discount_starts_at TIMESTAMPTZ,
      ADD COLUMN discount_ends_at   TIMESTAMPTZ;
  ```

  `indirimAktif()` tek yerde tanımlı olduğu için (§3.3) koşula iki madde
  eklenir; ürün kartı, `/indirimli` sayfası ve sipariş fiyatlaması otomatik
  uyar. Veri taşıma gerekmez, bu turda yazılan kod yeniden yazılmaz.
- **Toplu stok içe aktarma (CSV)** — 40 ürünlük katalog için gereksiz.
- **Stok raporu/grafik** — `stock_movements` verisi duruyor, ihtiyaç olursa
  sonra eklenir.
