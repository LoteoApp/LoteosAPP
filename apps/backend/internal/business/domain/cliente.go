package domain

import "time"

var (
	ErrClienteNoEncontrado = &Error{Kind: KindNotFound, Code: "client_not_found", Message: "Cliente no encontrado"}
	ErrDNIEnUso            = &Error{Kind: KindConflict, Code: "dni_in_use", Message: "El DNI ya está en uso"}
	ErrClienteInvalido     = &Error{Kind: KindInvalid, Code: "invalid_client", Message: "Nombre, apellido y DNI son obligatorios"}
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
