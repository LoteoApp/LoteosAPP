package domain

import (
	"strings"
	"time"
)

var (
	ErrAgencyNotFound    = &Error{Kind: KindNotFound, Code: "agency_not_found", Message: "Inmobiliaria no encontrada"}
	ErrCUITInUse         = &Error{Kind: KindConflict, Code: "cuit_in_use", Message: "El CUIT ya está en uso"}
	ErrInvalidAgency     = &Error{Kind: KindInvalid, Code: "invalid_agency", Message: "La razón social es obligatoria"}
	ErrInvalidAgencyID   = &Error{Kind: KindInvalid, Code: "invalid_agency_id", Message: "ID de inmobiliaria inválido"}
	ErrEmptyAgencyUpdate = &Error{Kind: KindInvalid, Code: "empty_agency_update", Message: "No se enviaron campos para modificar"}
	ErrInvalidCUIT       = &Error{Kind: KindInvalid, Code: "invalid_cuit", Message: "El CUIT debe tener 11 dígitos"}
)

const cuitDigits = 11

// Agency is an inmobiliaria: an external real-estate agency associated with
// loteos. The JSON tags keep the Spanish names the API already publishes.
type Agency struct {
	ID            string     `json:"id"`
	BusinessName  string     `json:"razonSocial"`
	CUIT          *string    `json:"cuit,omitempty"`
	Phone         *string    `json:"telefono,omitempty"`
	Email         *string    `json:"email,omitempty"`
	ModifiedBy    string     `json:"-"`
	DeactivatedAt *time.Time `json:"-"`
	CreatedAt     time.Time  `json:"fechaCreacion"`
	UpdatedAt     time.Time  `json:"fechaModificacion"`
}

// AgencyUpdate carries the fields a caller wants to change on an existing
// Agency for a PATCH-style update. A nil field means "leave this field
// unchanged"; a non-nil one replaces the stored value. Clearing an optional
// field back to null isn't supported by this type.
type AgencyUpdate struct {
	ID           string
	BusinessName *string
	CUIT         *string
	Phone        *string
	Email        *string
	ModifiedBy   string
}

// NormalizeCUIT drops the separators a CUIT is usually typed with, so
// "30-71234567-8" and "30712345678" are stored as the same value and the
// unique index on active inmobiliarias actually catches the duplicate.
func NormalizeCUIT(cuit string) string {
	var digits strings.Builder
	for _, r := range cuit {
		if r == '-' || r == ' ' || r == '.' {
			continue
		}
		digits.WriteRune(r)
	}

	return digits.String()
}

// ValidCUIT reports whether cuit is 11 digits once normalized. The check
// digit is not verified: the goal is to reject typos and free text, not to
// validate against AFIP.
func ValidCUIT(cuit string) bool {
	if len(cuit) != cuitDigits {
		return false
	}
	for _, r := range cuit {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
