package organizations_test

import (
	"strings"
	"testing"

	"pulso/internal/organizations"
)

func TestNewStatusPageSlug(t *testing.T) {
	first := organizations.NewStatusPageSlug("Ada Lovelace", "11111111-1111-1111-1111-111111111111")
	second := organizations.NewStatusPageSlug("Ada Lovelace", "22222222-2222-2222-2222-222222222222")
	if first != "ada-lovelace-11111111" || second != "ada-lovelace-22222222" {
		t.Fatalf("slugs = %q, %q", first, second)
	}
	if first == second {
		t.Fatal("slugs are not unique")
	}
	if !organizations.ValidStatusSlug(first) {
		t.Fatalf("generated slug is invalid: %q", first)
	}
}

func TestValidStatusSlug(t *testing.T) {
	valid := []string{"abc", "my-org-123", "a1b2"}
	invalid := []string{"", "ab", "-abc", "abc-", "ABC", "with space", "with_underscore", strings.Repeat("a", 65)}
	for _, slug := range valid {
		if !organizations.ValidStatusSlug(slug) {
			t.Errorf("expected valid slug %q", slug)
		}
	}
	for _, slug := range invalid {
		if organizations.ValidStatusSlug(slug) {
			t.Errorf("expected invalid slug %q", slug)
		}
	}
}
