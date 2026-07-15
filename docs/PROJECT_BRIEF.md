# PROJE: Çiçekçi Sipariş Sitesi — Faz 1 (MVP)

## Bağlam

Bir çiçekçi esnafı için sipariş alma sitesi geliştiriyorum. MVP için düşünüdklerim aşağıda

## Teknoloji Stack

- **Backend:** Go (chi veya gin router, standart kütüphane ağırlıklı,
  minimal framework bağımlılığı)
- **Frontend:** Vue 3 (Composition API), mobile-first responsive tasarım
- **Veritabanı:** PostgreSQL
- **Auth:** Admin paneli için basit JWT tabanlı authentication (tek admin
  kullanıcı yeterli, çok kullanıcılı rol sistemi gerekmiyor şimdilik)

## Kapsam — Bu Fazda Yapılacaklar

### 1. Public site (müşteri tarafı)
- Ana sayfa: öne çıkan ürünler, kısa tanıtım, öne çıkan kategori kartları
- Ürün listeleme sayfası: kategoriye göre filtreleme (aşağıdaki kategori
  yapısına bakınız)
- Ürün detay sayfası: görsel(ler), açıklama, fiyat
- Hakkımızda ve İletişim sayfaları
- Her ürün detayında "WhatsApp'tan Sipariş Ver" butonu — tıklanınca
  wa.me linki ile önceden doldurulmuş mesaj açılıyor (ürün adı, link
  otomatik mesaja eklenmeli)

### 2. Admin panel (çiçekçi tarafı, /admin altında, login korumalı)
- Giriş ekranı (JWT auth)
- Ürün CRUD: ekle / düzenle / sil / listele
- Her ürün için: isim, açıklama, fiyat, kategori(ler), görsel(ler),
  stok durumu (basit boolean: "stokta var / yok")
- Kategori yönetimi: ekle/düzenle/sil (bkz. kategori yapısı)

### 3. Teknik gereksinimler
- Mobile-first responsive tasarım (önce dar ekran, sonra genişlet)
- Temel SEO: her sayfada meta title/description, sitemap.xml,
  semantik HTML yapısı
- Görsel optimizasyonu: yüklenen ürün görselleri sıkıştırılmalı/uygun
  boyuta getirilmeli (lazy loading düşün)
- Ortam değişkenleri (.env) ile config yönetimi (DB bağlantısı, JWT
  secret vb. — hardcode yok)

## Kategori Yapısı (many-to-many)

Ürünler birden fazla kategoriye atanabilmeli. İki ayrı kategori ekseni
olacak, bir ürün her iki eksenden de kategori(ler) alabilir:

### 1. Gönderim amacına göre (occasion_category)
- Doğum Günü
- Yıl Dönümü
- Sevgiliye Çiçek
- Geçmiş Olsun
- Yeni İş / Terfi
- Özür Çiçeği
- Taziye
- Düğün / Nişan / Söz
- Yeni Bebek
- Kadınlar Günü / Anneler Günü / Öğretmenler Günü (mevsimsel, admin
  panelden eklenip çıkarılabilir olmalı — özel gün kategorileri
  dönemsel)

### 2. Ürün tipine göre (product_type_category)
- Buket
- Aranjman (vazoda/sepette düzenleme)
- Kutuda Çiçek
- Saksı Çiçeği
- Orkide
- Gelin Çiçeği / Yaka Çiçeği

> Not: Bu liste bir başlangıç iskeleti. Müşteriyle görüşüp gerçek ürün
> yelpazesine göre kesinleştirilmeli (örn. bazı çiçekçiler "Ferforje
> Çiçek" ya da "Teraryum" gibi niş ürünler de satabiliyor).

## Veritabanı Şeması Notu

- `products` tablosu: id, name, description, price, stock_status
  (boolean), created_at, updated_at
- `categories` tablosu: id, name, axis (enum: 'occasion' | 'type')
- `product_categories` (junction tablosu): product_id, category_id
- Admin panelde kategori ekle/düzenle/sil ayrı bir ekran olmalı, ürün
  eklerken bu kategorilerden çoklu seçim yapılabilmeli (multi-select
  veya checkbox listesi)

## Frontend Notu

- Ürün listeleme sayfasında iki ayrı filtre grubu olmalı: "Gönderim
  Amacına Göre" ve "Ürün Tipine Göre" — kullanıcı ikisinden de seçip
  kombinasyon filtreleme yapabilmeli (örn. "Doğum Günü" + "Buket"
  seçince ikisine de uyan ürünler listelensin)

## Mimari Notlar

- Backend'de ürün/kategori modellerini, Faz 2'de sepet/sipariş
  tablolarının kolayca eklenebileceği şekilde tasarla (örn. ürünlerin
  id yapısı, ileride order_items gibi bir tabloyla ilişkilendirilebilir
  olmalı)
- Admin panel routing'i, ileride sipariş yönetimi sayfası eklenecekmiş
  gibi genişletilebilir tut (örn. /admin/orders için hazır bir yer
  bırak, şimdilik boş/placeholder olabilir)
- API'yi REST prensipleriyle kur, ileride mobil app ihtimaline karşı
  backend'i frontend'den bağımsız, temiz bir şekilde ayır

## Çalışma Şekli

1. Önce backend'in temel API'sini (ürün/kategori CRUD + auth) kur ve
   test et
2. Sonra frontend'i backend'e bağla
3. Admin panel ve public site'ı ayrı route grupları olarak organize et
4. Adım adım ilerle, her parçayı tamamlayıp test ettikten sonra
   diğerine geç

---

## Sonraki Fazlar

**Faz 2 — Sipariş yönetimi ve teslimat planlaması**
- Site içi sepet (ödeme yok, sepeti WhatsApp mesajına dönüştürme)
- Teslimat tarihi/saat aralığı seçimi, kart mesajı alanı
- Admin panelde sipariş listesi
- Stok/durum takibi

**Faz 3 — Gerçek ödeme entegrasyonu**
- iyzico/PayTR entegrasyonu
- Sipariş durumu otomatik güncelleme
- İade/iptal akışı
- Not: Bu faza geçildiğinde ETBİS kaydı gerekecek (kendi web sitesi
  üzerinden doğrudan satış yapan işletmeler için zorunlu)

**Faz 4 — Büyüme ve optimizasyon (opsiyonel, talebe göre)**
- Gelir/sipariş raporlama paneli
- SEO derinleştirme (blog, bölge bazlı sayfalar — "izmir çiçekçi" gibi
  yerel aramalar için)
- Kampanya/indirim kodu sistemi
- Müşteri hesabı/sipariş geçmişi
- E-posta/SMS bildirimleri
