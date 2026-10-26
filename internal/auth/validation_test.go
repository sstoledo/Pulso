package auth_test

import (
	"errors"
	"testing"

	"pulso/internal/auth"
)

func TestValidateRegistrationNormalizesIdentity(t *testing.T) {
	input, err := auth.ValidateRegistration("  Ada Lovelace  ", "  ADA@EXAMPLE.COM ", "password-12345")
	if err != nil {
		t.Fatalf("ValidateRegistration() error = %v", err)
	}
	if input.Name != "Ada Lovelace" || input.Email != "ada@example.com" {
		t.Fatalf("normalized input = %#v", input)
	}
	if input.Password != "password-12345" {
		t.Fatal("password was unexpectedly changed")
	}
}

func TestValidateRegistrationRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name     string
		userName string
		email    string
		password string
		field    string
	}{
		{name: "empty name", email: "ada@example.com", password: "password-12345", field: "name"},
		{name: "invalid email", userName: "Ada", email: "not-an-email", password: "password-12345", field: "email"},
		{name: "short password", userName: "Ada", email: "ada@example.com", password: "short", field: "password"},
		{name: "long password", userName: "Ada", email: "ada@example.com", password: string(make([]byte, 129)), field: "password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.ValidateRegistration(tt.userName, tt.email, tt.password)
			var validation *auth.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("error = %v, want ValidationError", err)
			}
			if validation.Fields[tt.field] == "" {
				t.Fatalf("fields = %#v, missing %q", validation.Fields, tt.field)
			}
		})
	}
}
