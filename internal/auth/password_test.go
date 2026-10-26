package auth_test

import (
	"strings"
	"testing"

	"pulso/internal/auth"
)

func TestPasswordHashVerification(t *testing.T) {
	hash, err := auth.HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Fatalf("hash does not identify Argon2id version 19: %q", hash)
	}
	if strings.Contains(hash, "correct-horse-battery") {
		t.Fatal("hash contains the plaintext password")
	}

	valid, err := auth.VerifyPassword(hash, "correct-horse-battery")
	if err != nil || !valid {
		t.Fatalf("correct password: valid = %v, err = %v", valid, err)
	}
	valid, err = auth.VerifyPassword(hash, "incorrect-password")
	if err != nil || valid {
		t.Fatalf("incorrect password: valid = %v, err = %v", valid, err)
	}
}

func TestPasswordHashesUseUniqueSalts(t *testing.T) {
	first, err := auth.HashPassword("same-password-123")
	if err != nil {
		t.Fatal(err)
	}
	second, err := auth.HashPassword("same-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("equal passwords produced equal hashes")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	if _, err := auth.VerifyPassword("not-an-argon2-hash", "password-123"); err == nil {
		t.Fatal("expected malformed hash error")
	}
}
