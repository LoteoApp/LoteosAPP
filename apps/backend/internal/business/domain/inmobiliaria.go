package domain

import (
	"strings"
	"time"
)

var (
	ErrInmobiliariaNoEncontrada = &Error{Kind: KindNotFound, Code: "agency_not_found", Message: "Inmobiliaria no encontrada"}
	ErrCUITEnUso                = &Error{Kind: KindConflict, Code: "cuit_in_use", Message: "El CUIT ya está en uso"}
	ErrInmobiliariaInvalida     = &Error{Kind: KindInvalid, Code: "invalid_agency", Message: "La razón social es obligatoria"}
	ErrInmobiliariaIDInvalido   = &Error{Kind: KindInvalid, Code: "invalid_agency_id", Message: "ID de inmobiliaria inválido"}
	ErrInmobiliariaSinCambios   = &Error{Kind: KindInvalid, Code: "empty_agency_update", Message: "No se enviaron campos para modificar"}
	ErrCUITInvalido             = &Error{Kind: KindInvalid, Code: "invalid_cuit", Message: "El CUIT debe tener 11 dígitos"}
)

const cuitDigits = 11

type Inmobiliaria struct {
	ID                  string     `json:"id"`
	RazonSocial         string     `json:"razonSocial"`
	CUIT                *string    `json:"cuit,omitempty"`
	Telefono            *string    `json:"telefono,omitempty"`
	Email               *string    `json:"email,omitempty"`
	UsuarioModificacion string     `json:"-"`
	FechaBaja           *time.Time `json:"-"`
	FechaCreacion       time.Time  `json:"fechaCreacion"`
	FechaModificacion   time.Time  `json:"fechaModificacion"`
}

// InmobiliariaUpdate carries the fields a caller wants to change on an
// existing Inmobiliaria for a PATCH-style update. A nil field means "leave
// this field unchanged"; a non-nil one replaces the stored value. Clearing
// an optional field back to null isn't supported by this type.
type InmobiliariaUpdate struct {
	ID                  string
	RazonSocial         *string
	CUIT                *string
	Telefono            *string
	Email               *string
	UsuarioModificacion string
}

// NormalizarCUIT drops the separators a CUIT is usually typed with, so
// "30-71234567-8" and "30712345678" are stored as the same value and the
// unique index on active inmobiliarias actually catches the duplicate.
func NormalizarCUIT(cuit string) string {
	var digits strings.Builder
	for _, r := range cuit {
		if r == '-' || r == ' ' || r == '.' {
			continue
		}
		digits.WriteRune(r)
	}

	return digits.String()
}

// CUITValido reports whether cuit is 11 digits once normalized. The check
// digit is not verified: the goal is to reject typos and free text, not to
// validate against AFIP.
func CUITValido(cuit string) bool {
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
