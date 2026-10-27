package auth_test

import (
	"strings"
	"testing"

	"pulso/internal/auth"
)

func TestValidateRegistrationPasswordCharacterBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "minimum Unicode characters", password: strings.Repeat("é", 12)},
		{name: "maximum Unicode characters", password: strings.Repeat("é", 128)},
		{name: "over maximum Unicode characters", password: strings.Repeat("é", 129), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.ValidateRegistration("Ada", "ada@example.com", tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateRegistration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
