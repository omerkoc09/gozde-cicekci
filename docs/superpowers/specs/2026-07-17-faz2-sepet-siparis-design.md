# Faz 2 — Sepet ve Sipariş Tasarım Dokümanı

Tarih: 2026-07-17
Durum: Onaylandı, implementasyon planı bekliyor
Kaynak: `docs/superpowers/specs/2026-07-15-cicekci-mvp-design.md` (§9 Faz 2) üzerine
yapılan tasarım tartışması

---

## 1. Amaç ve Kapsam

Müşterinin birden fazla ürünü sepete atıp tek seferde sipariş verebilmesi; esnafın
gelen siparişleri admin panelde görüp işleyebilmesi.

**Sepet, Faz 3'teki ödemenin temelidir.** Bugün ödeme yok ama sepet ve sipariş
kaydı ona hazır kurulur — Faz 3'te sepeti baştan yazmak yerine üstüne ödeme adımı
eklenir.

### Yayın stratejisi — MVP spec'inden SAPMA

MVP spec'i fazları sırayla yayınlanan sürümler gibi kurgulamıştı ("Faz 1'de site
üzerinden ödeme alınmaz, müşteri WhatsApp'a yönlendirilir"). **Kullanıcı kararı:
site, ödeme sistemi hazır olmadan canlıya açılmayacak.** Gerekçe: ödeme olmadan
ürün teslim edilmeyecek, dolayısıyla ödemesiz bir site çalışmaz.

Sonuçları:
- Faz 2 ve Faz 3 ayrı **geliştirilir**, birlikte **yayınlanır**.
- Faz 2'de gerçek müşteri/sipariş olmayacağı için bildirim (mail/SMS) gerekmez.
- Deployment aciliyeti yok; altyapı hazır (`DEPLOYMENT.md`), ödeme bitince çıkılır.

### Bu fazda var
- Site içi sepet (localStorage), sepet drawer'ı gerçek hale gelir
- Sipariş formu: sipariş veren + alıcı + teslimat + kart mesajı
- Sunucuda sipariş kaydı (`orders`, `order_items`), fiyat doğrulaması
- Admin panelde sipariş listesi, detay, durum yönetimi

### Bu fazda yok (bilinçli)

| Ne | Neden | Nereye |
|---|---|---|
| Ödeme entegrasyonu | Sepet ona hazır kuruluyor | Faz 3 |
| Üyelik / `customers` | Üyenin yapacağı bir şey yok — MVP §3.3 | Faz 3 |
| Stok takibi | Ödeme kararıyla birlikte tasarlanmalı — MVP §3.2 | Faz 3 |
| Bildirim (mail/SMS) | Faz 2'de gerçek sipariş yok; Faz 3'te ödeme sağlayıcısının maili sinyal verir | Faz 3 |
| Bölge bazlı teslimat ücreti | Config yetiyor | `settings` tetiklenirse (MVP §8) |
| Slot kapasitesi / doluluk | Bugün ihtiyaç yok | Faz 4 |
| Terk edilmiş sepet | Üyelik gerektirir | Faz 4 |

### Neden üyelik bu fazda değil

Kullanıcı üyelik istedi, tartışıldı, **Faz 3'e ertelendi**. Gerekçe MVP §3.3 ile
aynı ve hâlâ geçerli: üyelik bir kapıdır, arkasında oda olmalı. Bugün hesap açan
müşteri hesabında hiçbir şey göremez — sipariş geçmişi yok (sipariş yeni
geliyor), kayıtlı kart yok (ödeme yok). Doğru sıra: önce sipariş var olsun,
sonra üyelik onu sahiplensin. Faz 3'te `orders.customer_id` nullable eklenir,
mevcut şemaya dokunulmaz.

---

## 2. Mimari Kararlar

### 2.1 Sepet tarayıcıda, sipariş sunucuda

Sepet `localStorage`'da yaşar. Sunucuda `carts` tablosu **yok**.

**Neden:** Üyelik olmadan sunucu sepeti ancak anonim session ile tutulabilir;
karşılığı zayıf (çiçek alımı yılda 2-3 kez, tek oturumda biter — MVP §3.3),
maliyeti somut (tablo, session yönetimi, terk edilmiş sepet temizliği, KVKK).
localStorage sepeti sıfır backend maliyeti getirir.

**Kabul edilen bedel:** Sepet cihazlar arası taşınmaz. Bu ölçekte sorun değil.

### 2.2 Fiyata asla güvenilmez

Sepetteki fiyat **gösterim içindir, sözleşme değildir.** Sipariş oluşurken sunucu
her ürünün fiyatını DB'den yeniden okur.

**Neden:** localStorage istemci tarafındadır; müşteri 1.850₺'yi 1₺ yapabilir.
Bu, tasarımın en kritik güvenlik kuralı.

### 2.3 Sepet ve WhatsApp birlikte yaşar

İki ayrı yol, ikisi de kalır:
- **WhatsApp butonu** — tek ürün, hızlı, pazarlığa açık. Mevcut haliyle korunur.
- **Sepet** — çok ürün, ödemeye giden yol (Faz 3).

Luxe redesign spec'i §2.3: *"WhatsApp tamamen kaldırılırsa site demo değil kırık
olur."* Bu geçerliliğini koruyor.

### 2.4 Sipariş otomatik düşer, durum elle ilerler

Müşteri "Siparişi Tamamla"ya bastığı an sunucu `orders` kaydını `pending` olarak
oluşturur — esnafın müdahalesi yok. Sonraki durumlar (`confirmed`, `delivered`,
`cancelled`) esnaf tarafından panelden elle değiştirilir.

**Bilinen risk:** MVP §3.2, stok için "esnaf güncellemeyi bırakır, bilgi yalan
söylemeye başlar" uyarısı yapmıştı. Aynı risk statüler için de var. Fark: esnaf
statüye **kendi işini takip etmek için** dokunuyor (stokta ise site için
dokunuyordu, karşılığı yoktu). Riski azaltmak için statü sayısı minimumda
tutuluyor: dört statü, iki tıklama.

---

## 3. Veri Modeli

```sql
orders
  id                BIGSERIAL PK
  order_no          TEXT NOT NULL UNIQUE   -- müşteriye söylenen no, ör. "2607-0042"
  status            TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','confirmed','delivered','cancelled'))

  -- sipariş veren (ödeyen)
  buyer_name        TEXT NOT NULL
  buyer_phone       TEXT NOT NULL
  buyer_email       TEXT                   -- opsiyonel

  -- alıcı (çiçek kime gidiyor)
  recipient_name    TEXT NOT NULL
  recipient_phone   TEXT NOT NULL
  delivery_address  TEXT NOT NULL
  delivery_date     DATE NOT NULL
  delivery_slot     TEXT NOT NULL          -- "12:00-15:00"
  card_message      TEXT                   -- opsiyonel

  -- tutarlar (sunucu hesaplar)
  items_total       NUMERIC(10,2) NOT NULL
  delivery_fee      NUMERIC(10,2) NOT NULL
  total             NUMERIC(10,2) NOT NULL

  note              TEXT                   -- esnafın kendi notu
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()

order_items
  id                BIGSERIAL PK
  order_id          BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE
  product_id        BIGINT REFERENCES products(id) ON DELETE SET NULL
  product_name      TEXT NOT NULL          -- sipariş anındaki ad (kopya)
  price_at_order    NUMERIC(10,2) NOT NULL -- sipariş anındaki fiyat (kopya)
  quantity          INT NOT NULL CHECK (quantity > 0)
```

Index: `orders(status, created_at DESC)` — panel listesi bu sırayla okuyor.

### Şema kararlarının gerekçeleri

- **`price_at_order` ve `product_name` kopya.** MVP §4.1 zaten uyarmış: sipariş
  anındaki fiyat bugünküyle aynı olmayacak. Esnaf yarın fiyat değiştirince dünkü
  sipariş bozulmamalı. Ad da aynı sebeple kopyalanıyor.

- **`product_id` nullable + `ON DELETE SET NULL`.** Ürün silinirse sipariş
  kaydı ölmemeli; ad/fiyat kopya olduğu için sipariş okunabilir kalır.

- **`order_no` ayrı kolon.** `id` yerine müşteriye söylenecek okunabilir numara.
  `id` ayrıca kaç sipariş alındığını dışarı sızdırır.

  **Format:** `AAGG-NNNN` — ay+gün, tire, o günün sıra numarası (`0001`'den
  başlar, her gün sıfırlanır). Örnek: 26 Temmuz'un 42. siparişi → `2607-0042`.
  Esnaf "bugünün 42. siparişi" diye okuyabilir.

  **Eşzamanlılık:** İki sipariş aynı anda gelirse aynı numarayı görebilir.
  `UNIQUE` constraint bunu engeller; store katmanı çakışmada bir sonraki sırayı
  deneyip tekrarlar (sınırlı retry). Aynı sorun Plan 1'de `uniqueSlug`'da
  yaşandı — orada DB constraint engelledi ama hata kaba düştü. Burada retry ile
  düzgün ele alınır. Tek esnaf ölçeğinde pratik risk düşük, yine de doğru
  yapılır.

- **`buyer` / `recipient` ayrımı.** Çiçekte alıcı ile gönderen farklı kişiler
  (MVP §3.3: *"çiçekte ürün hep başkasına gider"*). Kurye alıcıyı arayamazsa
  teslimat başarısız olur — `recipient_phone` operasyonel zorunluluk.

- **`customers` tablosu yok.** Bkz. §1. Faz 3'te `orders.customer_id` nullable
  eklenir.

- **`NUMERIC(10,2)`.** MVP §4.1 ile tutarlı: para float'ta tutulmaz.

- **`delivery_fee` siparişe kopyalanır.** `price_at_order` ile aynı gerekçe:
  esnaf yarın ücreti 50→60 yaparsa dünkü siparişin toplamı değişmemeli. Sipariş
  anında config'den okunup kayda yazılır; sonraki okumalarda config'e bakılmaz.

- **`items_total`, `delivery_fee`, `total` üçü de saklanır.** `total`
  türetilebilir ama saklanıyor: sipariş bir finansal kayıt, okunurken hesap
  yapmak yerine ne yazdıysa o olmalı. Tutarsızlık ihtimaline karşı tutarlar
  sipariş oluşturulurken tek yerde (service) hesaplanır.

---

## 4. API

### Public (auth yok)

```
POST /api/orders            → sipariş oluştur
GET  /api/delivery-config   → ücret, saat dilimleri, cutoff, max gün
```

`POST /api/orders` gövdesi **fiyat içermez** — sadece ürün id + adet:

```json
{
  "items": [{ "product_id": 3, "quantity": 2 }],
  "buyer": { "name": "...", "phone": "...", "email": "..." },
  "recipient": { "name": "...", "phone": "..." },
  "delivery": { "address": "...", "date": "2026-07-20", "slot": "12:00-15:00" },
  "card_message": "..."
}
```

Yanıt: `{ "order_no": "2607-0042", "total": "3750.00" }`

`GET /api/delivery-config` **neden ayrı uç:** Sunucu ve frontend aynı kaynaktan
beslenmeli. Frontend `.env`'i ayrı okursa 50₺ gösterip sunucu 60₺ hesaplayabilir.

### Admin (JWT korumalı)

```
GET   /api/admin/orders          ?status=&page=
GET   /api/admin/orders/{id}
PATCH /api/admin/orders/{id}     → status ve/veya note
```

Sipariş **oluşturma/silme admin ucu yok.** Siparişi müşteri oluşturur; esnaf
sadece durumunu değiştirir. Silme yok — sipariş finansal iz, `cancelled` var.

### Sunucu doğrulamaları (service katmanında)

| Kural | Neden |
|---|---|
| Ürün var ve `is_active` | Sepette dururken esnaf pasif yapmış olabilir |
| Fiyat DB'den okunur | localStorage'dan fiyat uydurulamasın |
| `delivery_date` ≥ bugün | Geçmişe sipariş olmaz |
| `delivery_date` ≤ bugün + `MAX_DELIVERY_DAYS` | Absürt ileri tarih |
| Aynı gün + şu an > cutoff → red | Esnaf yetiştiremez |
| `slot` config listesinde | Uydurma saat gelmesin |
| `quantity > 0` ve ≤ 1000 | Güvenlik duvarı — UI'da limit yok, bu sadece absürt girdiye karşı |
| Sepet boş değil | Boş sipariş olmaz |

**Atomiklik:** `orders` + `order_items` **tek transaction**. Yarısı yazılıp
kalanı patlarsa tutarsız sipariş kalır. (Plan 1'de slug'da aynı hata yaşandı,
probe testiyle kanıtlandı, tek transaction'a alındı — aynı ders.)

**Hata formatı** mevcutla aynı: `{"error": {"code": "...", "message": "..."}}`.

### Config (`.env`)

```
DELIVERY_FEE=50
DELIVERY_SLOTS=09:00-12:00,12:00-15:00,15:00-18:00
SAME_DAY_CUTOFF=16:00
MAX_DELIVERY_DAYS=30
```

**Neden config, `settings` tablosu değil:** MVP §5.3 ile aynı gerekçe — yılda bir
değişir, ayarlar ekranı YAGNI. **Değerler esnaftan öğrenilecek** (§7).

**Risk:** Esnaf "bölgeye göre değişiyor" derse config yetmez; `settings` veya
`delivery_zones` tablosu gerekir. Sonradan eklenebilir, baştan yapmanın kazancı yok.

---

## 5. Frontend

Luxe redesign spec'i §2.1: *"Bu ekranlar atılacak değil, backend'e bağlanacak
şekilde yazılır."* Şimdi o söz tutuluyor — kabuklar gerçek hale geliyor, tasarım
dili korunuyor.

**Sepet state:** `useCart()` composable, localStorage.
```ts
{ items: [{ product_id, name, slug, price, image, quantity }] }
```
Ürün bilgisi gösterim için tutulur; sipariş anında sunucu hepsini DB'den okur.

| Ne | Değişiklik |
|---|---|
| `TheCartDrawer` | Boş kabuk → gerçek ürünler, adet +/−, sil, ara toplam, "Siparişi Tamamla" |
| Sepet ikonu (`TheHeader`) | **Rozet geri geliyor** — artık gerçek sayı var |
| "Sepete Ekle" (ürün detay) | Inert → sepete ekler, drawer açılır |
| `WhatsAppButton` | **Dokunulmuyor** |
| `/siparis` | YENİ — sipariş formu |
| `/siparis/tamam` | YENİ — teşekkür sayfası, sipariş no |
| `/hesabim/*` | **Dokunulmuyor** — mock kalır, üyelik Faz 3 |
| Admin `/siparisler` | Placeholder → gerçek liste + detay + durum yönetimi |

**Rozet neden geri geliyor:** Redesign spec'i rozeti *"var olmayan sepet
içeriğini iddia etmesin"* diye kaldırmıştı. Sepet artık gerçek — rozet yalan
değil, bilgi. Spec'in kendi mantığı bunu gerektiriyor.

**Form uzunluğu riski:** MVP §3.3 uyarıyor — her adım dönüşümü düşürür.
Hafifletme: "Alıcı benim" kutucuğu (buyer bilgilerini kopyalar), tek sayfa (çok
adımlı sihirbaz değil), yalnız gerekli alanlar zorunlu (`buyer_email` opsiyonel).

**Kenar durumlar:**

- **Sepet boşken `/siparis`** → `/urunler`'e yönlendir. Boş form göstermek anlamsız.
- **Sipariş başarılı** → sepet temizlenir, `/siparis/tamam`'a gidilir. Temizlenmezse
  müşteri aynı siparişi ikinci kez verebilir.
- **Sipariş başarısız** (pasif ürün, fiyat değişmiş vb.) → sepet **korunur**,
  hata mesajı gösterilir. Müşterinin emeği silinmemeli.
- **Sepetteki ürün pasif olmuş/silinmiş** → sunucu reddeder; frontend hangi ürünün
  sorunlu olduğunu söyler ve sepetten çıkarmayı önerir.

**Sepette adet limiti yok** — 50 buket gerçek bir sipariş olabilir. Sunucudaki
1000 sınırı UI kısıtı değil, absürt girdiye karşı duvar.

---

## 6. Test Stratejisi

MVP §6 ile aynı mantık: `service`/`store` katmanında, HTTP kurmadan.

1. **Fiyat doğrulama** — en kritik. Sepette 1₺ gelse bile sunucu DB'den 1.850₺
   okumalı. Kırılırsa para kaybı.
2. **Sipariş atomikliği** — `orders` + `order_items` tek transaction; yarısı
   yazılıp patlarsa hiçbiri kalmamalı.
3. **Doğrulama kuralları** — pasif ürün reddi, geçmiş tarih reddi, cutoff sonrası
   aynı gün reddi, geçersiz slot reddi.

Frontend: `useCart` için vitest (ekle/çıkar/adet/toplam) — saf mantık.

Handler testleri az: çoğunlukla auth kontrolü (mevcut desen).

---

## 7. Esnafa Sorulacaklar

Bu değerler bilinmiyor; tasarım config'e dayanıyor, değerler öğrenilince girilecek.

- **Teslimat ücreti kaç?** Bölgeye göre değişiyor mu? (Değişiyorsa config yetmez,
  `settings`/`delivery_zones` gerekir — tasarım revize edilir.)
- **Hangi saat dilimleri?**
- **Aynı gün siparişte son saat kaç?**
- **"Teslim edildi" işaretlemesi yapar mı?** Yapmayacaksa `delivered` statüsü
  gereksiz — çıkarmak kolay.

---

## 8. Kabul Kriteri

Müşteri sepete iki ürün atar → form doldurur → sipariş oluşur → esnaf panelde
görür → onaylar → teslim edildi işaretler. Fiyat manipülasyonu sunucuda
reddedilir.

---

## 9. Faz 3'e Devreden

- **Ödeme entegrasyonu** — sepet buna hazır kuruldu.
- **MVP §3.2'nin açık sorusu:** "müşteri ödedi ama ürün yok". İki model
  tartışılacak: (a) provizyon — para bloke edilir, esnaf onaylayınca çekilir;
  (b) direkt çekim + stok takibi. Kullanıcı provizyon fikrine değindi, karar Faz
  3'te ödeme sağlayıcısı seçilirken verilecek. Bugünkü statü seti ikisini de
  destekler (`pending` ile `confirmed` arasına `paid`/`authorized` girer).
- **Üyelik** (`customers`, `orders.customer_id` nullable) — MVP §3.3.
- **Stok takibi** — ödeme modeliyle birlikte.
- **Bildirim** — ödeme sağlayıcısının maili yeterli mi, değerlendirilecek.
- **ETBİS kaydı** — kendi sitesinden satış yapan işletmeler için zorunlu
  (MVP §9). Kod işi değil, kullanıcının işi.
