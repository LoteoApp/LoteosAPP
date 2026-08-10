package domain

import (
	"errors"
	"time"
)

var (
	ErrEmailEnUso          = errors.New("email ya está en uso")
	ErrUsuarioNoEncontrado = errors.New("usuario no encontrado")
	ErrNoAutorizado        = errors.New("no autorizado")
	ErrEmailInvalido       = errors.New("email inválido")
	ErrRolInvalido         = errors.New("rol inválido")
	ErrPerfilInvalido      = errors.New("nombre y apellido son obligatorios")
)

type Usuario struct {
	ID             string    `json:"id"`
	KeycloakID     string    `json:"-"`
	Email          string    `json:"email"`
	Nombre         string    `json:"nombre"`
	Apellido       string    `json:"apellido"`
	Rol            string    `json:"rol"`
	PerfilCompleto bool      `json:"perfilCompleto"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Roles de dominio documentados en docs/domain.md, sección "Usuarios y roles".
const (
	RolAdministrador  = "administrador"
	RolAdministrativo = "administrativo"
	RolAgrimensor     = "agrimensor"
	RolEscribano      = "escribano"
	RolInmobiliaria   = "inmobiliaria"
)

var rolesValidos = map[string]struct{}{
	RolAdministrador:  {},
	RolAdministrativo: {},
	RolAgrimensor:     {},
	RolEscribano:      {},
	RolInmobiliaria:   {},
}

func RolValido(rol string) bool {
	_, ok := rolesValidos[rol]
	return ok
}
