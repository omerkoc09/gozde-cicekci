package customer

import "testing"

func TestGenerateParseToken_RoundTrip(t *testing.T) {
	secret := "test-secret-en-az-32-karakter-olmali-xx"
	tok, err := GenerateToken(42, secret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	claims, err := ParseToken(tok, secret)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.CustomerID != 42 {
		t.Fatalf("CustomerID = %d, beklenen 42", claims.CustomerID)
	}
	if claims.Type != "customer" {
		t.Fatalf("Type = %q, beklenen customer", claims.Type)
	}
}

func TestParseToken_YanlisSecretRed(t *testing.T) {
	tok, _ := GenerateToken(1, "secret-a-en-az-32-karakter-uzunlugunda-x")
	if _, err := ParseToken(tok, "secret-b-en-az-32-karakter-uzunlugunda-x"); err == nil {
		t.Fatal("yanlış secret reddedilmeliydi")
	}
}
