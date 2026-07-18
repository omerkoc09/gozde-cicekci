# Admin Kullanıcı Yönetimi Tasarım Dokümanı

Tarih: 2026-07-18
Durum: Onaylandı, implementasyon planı bekliyor

---

## 1. Amaç ve Kapsam

Şu an yeni admin eklemenin tek yolu CLI seed komutu (`make seed` / `backend/cmd/seed/main.go`),
sunucuya SSH veya CLI erişimi gerektiriyor. Amaç: admin panelden (frontend/idare)
yeni admin ekleyebilme, mevcut adminleri listeleyebilme, silebilme ve şifre
sıfırlayabilme.

Rol/yetki sistemi **eklenmiyor** — tüm adminler bugün olduğu gibi eşit yetkili
kalır (`frontend/idare/src/store/user.ts` içindeki mevcut karar: "tek admin var,
rol sistemi yok" ilkesi korunur). `admin_users` şeması değişmez, migration
gerekmez.

### Bu kapsamda var
- Admin listesi görüntüleme (kullanıcı adı)
- Yeni admin ekleme (kullanıcı adı + şifre, panelden)
- Admin silme
- Herhangi bir adminin şifresini sıfırlama (kendisininki dahil)

### Bu kapsamda yok (bilinçli)
| Ne | Neden |
|---|---|
| Rol/yetki ayrımı | Tüm adminler bugün eşit yetkili; ihtiyaç yok (YAGNI) |
| E-posta, profil bilgisi | `admin_users` şeması minimal kalıyor, kullanılmayacak alan eklenmiyor |
| Ayrı "profilim" sayfası | Kullanıcı listesindeki şifre sıfırlama aksiyonu kendi hesap için de kullanılabilir |
| CLI seed komutunun kaldırılması | İlk admin oluşturma (bootstrap) hâlâ CLI'a bağlı — panel sadece ek admin eklemek için var |

### Güvenlik kuralları
- **Kendini silemez**: Bir admin kendi hesabını silemez (yanlışlıkla kendini
  kilitleme riskini önler).
- **Son admin korunur**: Sistemde 1 admin kaldıysa silme işlemi reddedilir.
- **Şifre sıfırlama kısıtı yok**: Herhangi bir admin, herhangi bir adminin
  (kendisi dahil) şifresini sıfırlayabilir — şifresini unutan admin için pratik
  bir çözüm.

---

## 2. Backend Tasarımı

### Store katmanı (`backend/internal/auth/store.go`)

Mevcut `FindByUsername`, `Create` metodlarına ek olarak:

- `List(ctx) ([]AdminUser, error)` — tüm adminleri `id, username` ile döner
  (password_hash zaten `json:"-"`, ama query'de de gereksiz çekilmeyebilir)
- `Delete(ctx, id int64) error` — `DELETE FROM admin_users WHERE id = $1`
- `UpdatePassword(ctx, id int64, passwordHash string) error` — `UPDATE admin_users SET password_hash = $2 WHERE id = $1`
- `Count(ctx) (int, error)` — `SELECT COUNT(*) FROM admin_users`

### Service katmanı (`backend/internal/auth/service.go`)

Mevcut `CreateAdmin`, `Login` metodlarına ek olarak:

- `ListAdmins(ctx) ([]AdminUser, error)` — `Store.List` sarmalayıcı
- `DeleteAdmin(ctx, requesterID, targetID int64) error`:
  - `requesterID == targetID` → `errorsx.ErrInvalidInput` ("kendi hesabınızı silemezsiniz")
  - `Store.Count() <= 1` → `errorsx.ErrInvalidInput` ("son admin silinemez")
  - aksi halde `Store.Delete`
- `ChangePassword(ctx, id int64, newPassword string) error`:
  - `len(newPassword) < minPasswordLength` kontrolü (mevcut 8 karakter kuralı)
  - bcrypt hash + `Store.UpdatePassword`

### API katmanı

Yeni dosya `backend/internal/api/idare/user_handler.go`, `auth_handler.go`
deseninde (struct + fiber handler metodları):

- `GET /users` → `ListAdmins`, `[]{"id":.., "username":..}` döner
- `POST /users` → body `{username, password}`, `CreateAdmin` çağırır (mevcut
  validasyon: min 8 karakter, unique username → 409 conflict)
- `DELETE /users/:id` → `requesterID` JWT `Locals("userID")`'den okunur,
  `DeleteAdmin(requesterID, targetID)` çağırır
- `PATCH /users/:id/password` → body `{password}`, `ChangePassword` çağırır

`router.go`'daki `protected` grubuna eklenir:
```go
protected.Get("/users", uh.list)
protected.Post("/users", uh.create)
protected.Delete("/users/:id", uh.delete)
protected.Patch("/users/:id/password", uh.resetPassword)
```

`Deps` struct'ına ek alan gerekmez — `authHandler` zaten `d.AuthSvc` kullanıyor,
`userHandler` de aynı `auth.Service`'i paylaşır.

---

## 3. Frontend Tasarımı (frontend/idare)

Mevcut `kategoriler.vue` + `useCategories.ts` + `model/category.ts` üçlü deseni
taklit edilir.

- **Model** `src/model/adminUser.ts`: `{ id: number, username: string }`
- **Composable** `src/composables/useAdminUsers.ts`: `list()`, `create(username, password)`,
  `remove(id)`, `resetPassword(id, password)` — `ApiService` üzerinden `admin/users`
  çağrıları
- **Sayfa** `src/pages/kullanicilar.vue`:
  - `VDataTable` ile admin listesi (kullanıcı adı sütunu)
  - "Yeni Admin Ekle" butonu → dialog form (kullanıcı adı + şifre + şifre
    tekrar, min 8 karakter client-side validasyon)
  - Her satırda "şifre sıfırla" aksiyonu → dialog (yeni şifre gir, kaydet)
  - Her satırda "sil" aksiyonu → onay dialogu; giriş yapmış adminin kendi
    satırında sil butonu disabled (backend zaten reddeder, bu sadece UX için
    önden engelleme)
- **Navigasyon** `src/navigation/vertical/index.ts`'e "Kullanıcılar" linki eklenir

Kendi hesabının kimliği `me` endpoint'inden (`useAuthStore` / mevcut
`store/user.ts`) zaten biliniyor — sil butonunu disable etmek için kullanılır.

---

## 4. Hata durumları

| Durum | Backend yanıtı | Frontend gösterimi |
|---|---|---|
| Kullanıcı adı zaten var | 409 Conflict | "Bu kullanıcı adı zaten kullanılıyor" |
| Şifre < 8 karakter | 400 Bad Request | Form validasyonu (client-side önce yakalar) |
| Kendi hesabını silme denemesi | 400 Bad Request | Buton zaten disabled; API çağrılırsa hata mesajı gösterilir |
| Son admini silme denemesi | 400 Bad Request | "Son admin silinemez" mesajı |

---

## 5. Test planı

- Backend: `service_test.go`'ya `DeleteAdmin` (self-delete reddi, son-admin
  reddi, başarılı silme) ve `ChangePassword` (kısa şifre reddi, başarılı
  güncelleme) test case'leri
- Backend: mevcut `middleware_test.go` deseninde yeni route'ların auth
  koruması altında olduğunu doğrulayan test
- Frontend: mevcut proje genelinde otomatik test yok (kategoriler.vue için de
  yok) — manuel doğrulama ile yetinilir, mevcut pattern korunur
