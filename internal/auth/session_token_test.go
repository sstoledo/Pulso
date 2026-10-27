package auth_test

import (
	"bytes"
	"encoding/base64"
	"testing"

	"pulso/internal/auth"
)

func TestNewSessionTokenIsOpaqueAndHashed(t *testing.T) {
	token, hash, err := auth.NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken() error = %v", err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("token is not raw-base64url: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("decoded token length = %d, want 32", len(decoded))
	}
	if len(hash) != 32 {
		t.Fatalf("hash length = %d, want 32", len(hash))
	}
	if bytes.Contains(hash[:], []byte(token)) {
		t.Fatal("hash contains the raw token")
	}
	if got := auth.HashSessionToken(token); got != hash {
		t.Fatal("HashSessionToken returned a different hash")
	}

	otherToken, otherHash, err := auth.NewSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == otherToken || hash == otherHash {
		t.Fatal("session tokens or hashes were duplicated")
	}
}
