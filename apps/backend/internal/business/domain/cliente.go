package domain

import "time"

var (
	ErrClienteNoEncontrado = &Error{Kind: KindNotFound, Code: "client_not_found", Message: "Cliente no encontrado"}
	ErrDNIEnUso            = &Error{Kind: KindConflict, Code: "dni_in_use", Message: "El DNI ya está en uso"}
	ErrClienteInvalido     = &Error{Kind: KindInvalid, Code: "invalid_client", Message: "Nombre, apellido y DNI son obligatorios"}
	ErrClienteIDInvalido   = &Error{Kind: KindInvalid, Code: "invalid_client_id", Message: "ID de cliente inválido"}
	ErrClienteSinCambios   = &Error{Kind: KindInvalid, Code: "empty_client_update", Message: "No se enviaron campos para modificar"}
)

type Cliente struct {
	ID                  string     `json:"id"`
	Nombre              string     `json:"nombre"`
	Apellido            string     `json:"apellido"`
	DNI                 string     `json:"dni"`
	Celular             *string    `json:"celular,omitempty"`
	Email               *string    `json:"email,omitempty"`
	UsuarioModificacion string     `json:"-"`
	FechaBaja           *time.Time `json:"-"`
	FechaCreacion       time.Time  `json:"fechaCreacion"`
	FechaModificacion   time.Time  `json:"fechaModificacion"`
}

// ClienteUpdate carries the fields a caller wants to change on an existing
// Cliente for a PATCH-style update. A nil field means "leave this field
// unchanged" — that is what makes PATCH semantics correct: omitting a field
// in the request body must not clobber it. A non-nil field, including an
// empty string, replaces the existing value. Explicitly clearing Celular or
// Email back to null isn't supported by this type; that's a possible future
// extension, not something the current API needs.
type ClienteUpdate struct {
	ID                  string
	Nombre              *string
	Apellido            *string
	DNI                 *string
	Celular             *string
	Email               *string
	UsuarioModificacion string
}
