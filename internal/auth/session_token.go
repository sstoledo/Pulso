package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const sessionTokenLength = 32

func NewSessionToken() (string, [sha256.Size]byte, error) {
	var empty [sha256.Size]byte
	random := make([]byte, sessionTokenLength)
	if _, err := rand.Read(random); err != nil {
		return "", empty, fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(random)
	return token, HashSessionToken(token), nil
}

func HashSessionToken(token string) [sha256.Size]byte {
	return sha256.Sum256([]byte(token))
}
