# Deployment — Tek VPS

Bu rehber, projeyi tek bir VPS'e Docker Compose ile canlıya almak içindir.
Mimari: tek domain, Caddy reverse proxy (otomatik TLS), Go backend + Nuxt SSR
public site + admin panel (statik, `/idare` alt yolu) + Postgres, hepsi aynı
sunucuda. Görseller local diskte (volume), günlük yedek alınıyor.

Tüm bu yapı lokalde prod moduyla (self-signed TLS) uçtan uca test edildi.

---

## Mimari

```
İnternet ──▶ Caddy (:443, otomatik TLS)
              ├─ /            → Nuxt SSR public site (app:3000)
              ├─ /idare/*     → admin panel statik dosyalar (Caddy servis eder)
              ├─ /api/*       → Go backend (backend:8080)
              └─ /uploads/*   → Go backend (görsel dosyaları)

Nuxt ──(iç ağ, sunucu-sunucu)──▶ backend   (CORS yok, same-origin proxy)
backend ──▶ postgres (iç ağ)
postgres + uploads ──▶ backup (günlük pg_dump + tar)
```

Tek origin olduğu için CORS derdi yok: admin panel `/api`'yi, public site
kendi Nitro proxy'sini aynı origin'den kullanıyor.

---

## Ön koşullar

- Bir VPS (Hetzner / DigitalOcean / Contabo — 2 vCPU / 4GB yeter, hatta 2GB).
- Ubuntu 22.04+ veya benzeri.
- Bir domain (örn. `cicekci.com`) ve DNS'i VPS'in IP'sine yönlendirme yetkisi.
- Sunucuda Docker + Docker Compose kurulu.

Docker kurulumu (Ubuntu):
```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER   # tekrar login gerektirir
```

---

## Adım adım

### 1. Domain DNS'ini yönlendir

Domain sağlayıcının panelinde bir **A kaydı** ekle:

```
A   @   <VPS-IP-adresi>
```

(İstersen `www` için de aynı IP'ye A kaydı; Caddyfile'a `www.cicekci.com`
eklemen gerekir.) DNS yayılması birkaç dakika–saat sürebilir. Caddy TLS
sertifikasını alabilmesi için bu kaydın çalışıyor olması ŞART.

Kontrol: `dig +short cicekci.com` → VPS IP'sini dönmeli.

### 2. Kodu sunucuya al

```bash
git clone <repo-url> cicekci
cd cicekci
git checkout feat/backend-temeli   # veya merge sonrası main
```

### 3. Prod ortam değişkenlerini hazırla

```bash
cp .env.prod.example .env.prod
```

`.env.prod`'u düzenle ve GERÇEK değerleri gir. Kritik olanlar:

```bash
# Güçlü secret'lar üret:
openssl rand -base64 32   # → JWT_SECRET
openssl rand -base64 24   # → POSTGRES_PASSWORD
```

- `SITE_DOMAIN` = `cicekci.com` (TLS bunun için alınır)
- `SITE_URL` = `https://cicekci.com` (https ZORUNLU — backend kontrol ediyor)
- `POSTGRES_PASSWORD` = üretilen güçlü şifre
- `DATABASE_URL` = aynı şifreyle tutarlı olmalı (host `postgres`, port `5432`)
- `JWT_SECRET` = üretilen 32+ karakter anahtar
- `WHATSAPP_NUMBER` = siparişlerin düşeceği gerçek numara (ülke kodlu)
- `CONTACT_*` = iletişim sayfası bilgileri

> `.env.prod` `.gitignore`'da — asla commit edilmez.

### 4. Ayağa kaldır

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

İlk build birkaç dakika sürer (üç imaj derleniyor). Sıralama otomatik:
postgres hazır olunca → migrate çalışır → backend başlar → app + caddy başlar.
Caddy ilk çalışmada Let's Encrypt'ten sertifika alır (DNS doğruysa saniyeler).

Durumu kontrol et:
```bash
docker compose -f docker-compose.prod.yml ps
# migrate ve idare "Exited (0)" olmalı (one-shot), gerisi "Up".
```

### 5. İlk admin kullanıcısını oluştur

Panel için bir admin gerekli. Backend container'ında etkileşimsiz seed:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod \
  exec backend /app/seed -username <kullanici> -password <guclu-sifre>
```

(Şifre en az 8 karakter. Bu bilgileri güvenli sakla — panel girişi bununla.)

Alternatif, etkileşimli (şifre ekranda görünmez):
```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod exec backend /app/seed
```

### 6. Doğrula

Tarayıcıda:
- `https://cicekci.com` → public site açılmalı (TLS yeşil kilit)
- `https://cicekci.com/idare/` → admin panel, giriş ekranı
- Panelden giriş yap, bir kategori + ürün oluştur, fotoğraf yükle
- Public site'ta ürün görünmeli, WhatsApp butonu çalışmalı
- Ürün linkini WhatsApp'ta paylaş → fotoğraf önizlemesi çıkmalı (og:image)

Komut satırından:
```bash
curl -I https://cicekci.com/                    # 200
curl -I https://cicekci.com/idare/              # 200
curl -s https://cicekci.com/api/products        # [] veya ürünler
curl -s https://cicekci.com/urun/<slug> | grep og:image   # dolu olmalı
```

---

## Güncelleme (yeni sürüm deploy)

```bash
cd cicekci
git pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

Migration varsa migrate servisi otomatik çalıştırır. Kesinti minimum
(saniyeler) — Caddy eski container'ı yenisiyle değiştirir.

---

## Yedekleme ve geri yükleme

`backup` servisi her 24 saatte bir `./backups/` altına yazar (7 günden
eskiyi siler):
- `db_<tarih>.dump` — pg_dump (custom format)
- `uploads_<tarih>.tar.gz` — görsel dosyaları

**Bu dosyaları sunucu dışına da kopyala** (örn. haftalık `rsync`/`scp` ile
başka bir yere veya bir object storage'a) — sunucu tamamen ölürse yedek de
onunla gitmesin.

Geri yükleme (DB):
```bash
# uploads:
docker compose -f docker-compose.prod.yml --env-file .env.prod \
  exec -T backend sh -c 'cd /app/uploads && tar xzf -' < backups/uploads_<tarih>.tar.gz

# DB:
docker compose -f docker-compose.prod.yml --env-file .env.prod \
  exec -T postgres pg_restore -U <user> -d <db> --clean < backups/db_<tarih>.dump
```

---

## Notlar ve bilinen sınırlar

- **Görseller local diskte.** Tek VPS için yeter. Çoklu sunucuya geçersen
  veya diski dert etmek istemezsen backend'de R2 sürücüsü hazır
  (`STORAGE_DRIVER=r2` + R2 env'leri) — spec §8'de bu geçiş planlı.
- **Proje adı izolasyonu.** Prod compose `name: cicekci-prod` kullanıyor,
  volume'ları ayrı isimli. Aynı sunucuda dev compose çalıştırmasan da bu,
  yanlışlıkla veri silinmesini önler.
- **Tek Postgres, tek makine.** Bu ölçekte (tek esnaf, 40-100 ürün) fazlasıyla
  yeterli. Managed DB'ye geçiş ileride bir env değişikliği.
- **Sertifika yenileme** Caddy tarafından otomatik — elle bir şey yapmana
  gerek yok.

---

## Lokalde prod'u test etmek (opsiyonel)

Domain almadan, prod yapısını lokalde self-signed TLS ile deneyebilirsin:

```bash
cat > .env.prod.local <<'EOF'
SITE_DOMAIN=localhost
SITE_URL=https://localhost
POSTGRES_USER=cicekci
POSTGRES_PASSWORD=local-test
POSTGRES_DB=cicekci
DATABASE_URL=postgres://cicekci:local-test@postgres:5432/cicekci?sslmode=disable
JWT_SECRET=lokal-test-en-az-32-karakterlik-anahtar
WHATSAPP_NUMBER=905551234567
CONTACT_PHONE=0555 123 45 67
CONTACT_ADDRESS=Test
CONTACT_HOURS=09:00-20:00
EOF

docker compose -f docker-compose.prod.yml -f docker-compose.prod.local.yml \
  --env-file .env.prod.local up -d --build

curl -k https://localhost/          # -k: self-signed sertifikayı kabul et
```

Kaldır: `docker compose -f docker-compose.prod.yml -f docker-compose.prod.local.yml down -v`
