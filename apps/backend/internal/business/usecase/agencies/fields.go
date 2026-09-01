package agencies

import (
	"strings"

	"loteosapp/backend/internal/business/domain"
)

// trimIfPresent trims s when it's present (non-nil), leaving an absent
// field (nil, meaning "unchanged") as nil.
func trimIfPresent(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	return &trimmed
}

// trimOptional trims an optional field, treating a blank value as absent so
// `"telefono": "  "` doesn't get stored as whitespace.
func trimOptional(s *string) *string {
	trimmed := trimIfPresent(s)
	if isBlank(trimmed) {
		return nil
	}
	return trimmed
}

// isBlank reports whether s is present but empty — i.e. the caller sent the
// field but it has no content, as opposed to not sending it at all.
func isBlank(s *string) bool {
	return s != nil && *s == ""
}

// normalizeOptionalCUIT strips the separators a CUIT is typed with and
// rejects anything that isn't 11 digits, so the stored value is the only
// shape the unique index has to compare.
func normalizeOptionalCUIT(cuit *string) (*string, error) {
	if cuit == nil {
		return nil, nil
	}

	normalized := domain.NormalizarCUIT(*cuit)
	if !domain.CUITValido(normalized) {
		return nil, domain.ErrCUITInvalido
	}

	return &normalized, nil
}

func validateOptionalEmail(email *string) error {
	if email == nil {
		return nil
	}
	if !domain.EmailValido(*email) {
		return domain.ErrEmailInvalido
	}

	return nil
}
