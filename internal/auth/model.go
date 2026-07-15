package auth

// AdminUser bir yönetici kullanıcısını temsil eder.
type AdminUser struct {
	ID           int64
	Username     string
	PasswordHash string
}
