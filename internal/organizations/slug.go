package organizations

import (
	"regexp"
	"strings"
)

var statusSlugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,62}[a-z0-9])?$`)

func NewStatusPageSlug(organizationName, organizationID string) string {
	base := slugify(organizationName)
	if base == "" {
		base = "org"
	}
	if len(base) > 55 {
		base = strings.TrimRight(base[:55], "-")
	}
	return base + "-" + organizationID[:8]
}

func ValidStatusSlug(slug string) bool {
	return len(slug) >= 3 && len(slug) <= 64 && statusSlugPattern.MatchString(slug)
}

func slugify(value string) string {
	var builder strings.Builder
	lastDash := false
	for _, character := range strings.ToLower(value) {
		alphaNumeric := character >= 'a' && character <= 'z' || character >= '0' && character <= '9'
		if alphaNumeric {
			builder.WriteRune(character)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
