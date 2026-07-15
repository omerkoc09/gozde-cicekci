package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/omerkoc/cicekci/internal/auth"
	"github.com/omerkoc/cicekci/pkg/config"
	"github.com/omerkoc/cicekci/pkg/database"
	"golang.org/x/term"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("veritabanı: %v", err)
	}
	defer pool.Close()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Admin kullanıcı adı: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("okuma: %v", err)
	}
	username = strings.TrimSpace(username)

	fmt.Print("Şifre (en az 8 karakter): ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalf("şifre okuma: %v", err)
	}
	fmt.Println()
	password := strings.TrimSpace(string(passwordBytes))

	svc := auth.NewService(auth.NewStore(pool), cfg.JWTSecret)
	if err := svc.CreateAdmin(ctx, username, password); err != nil {
		log.Fatalf("admin oluşturulamadı: %v", err)
	}

	fmt.Printf("Admin kullanıcısı oluşturuldu: %s\n", username)
}
