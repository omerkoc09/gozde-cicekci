package auth

// AdminUser bir yönetici kullanıcısını temsil eder.
type AdminUser struct {
	ID       int64
	Username string
	// PasswordHash asla JSON'a çıkmaz — bu struct kazara bir HTTP
	// yanıtına serialize edilirse hash sızmasın diye.
	PasswordHash string `json:"-"`
}
