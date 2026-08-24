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
	AuthProviderID string    `json:"-"`
	Email          string    `json:"email"`
	Nombre         string    `json:"nombre"`
	Apellido       string    `json:"apellido"`
	Rol            Rol       `json:"rol"`
	PerfilCompleto bool      `json:"perfilCompleto"`
	CreatedAt      time.Time `json:"createdAt"`
}
