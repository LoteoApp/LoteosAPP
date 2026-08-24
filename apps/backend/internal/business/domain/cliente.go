package domain

import (
	"errors"
	"time"
)

var (
	ErrClienteNoEncontrado = errors.New("cliente no encontrado")
	ErrClienteInvalido     = errors.New("nombre, apellido y dni son obligatorios")
	ErrDNIEnUso            = errors.New("dni ya está en uso")
)

type Cliente struct {
	ID                string    `json:"id"`
	Nombre            string    `json:"nombre"`
	Apellido          string    `json:"apellido"`
	DNI               string    `json:"dni"`
	Celular           string    `json:"celular"`
	Email             string    `json:"email"`
	FechaCreacion     time.Time `json:"fechaCreacion"`
	FechaModificacion time.Time `json:"fechaModificacion"`
}

// Permisos sobre clientes documentados en docs/domain.md, sección "Clientes".
var rolesGestionCliente = map[string]struct{}{
	RolAdministrador:  {},
	RolAdministrativo: {},
	RolInmobiliaria:   {},
}

// PuedeGestionarClientes indica si el rol puede listar, dar de alta y
// modificar clientes.
func PuedeGestionarClientes(rol string) bool {
	_, ok := rolesGestionCliente[rol]
	return ok
}
