package customer

import "time"

// Customer bir müşteri hesabını temsil eder.
type Customer struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
	// PasswordHash asla JSON'a çıkmaz — kazara serialize edilse bile sızmasın.
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
