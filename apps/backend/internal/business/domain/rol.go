package domain

// Rol es uno de los roles de dominio documentados en docs/domain.md,
// sección "Usuarios y roles".
type Rol string

const (
	RolAdministrador  = "administrador"
	RolAdministrativo = "administrativo"
	RolAgrimensor     = "agrimensor"
	RolEscribano      = "escribano"
	RolInmobiliaria   = "inmobiliaria"
)

var rolesValidos = map[Rol]struct{}{
	RolAdministrador:  {},
	RolAdministrativo: {},
	RolAgrimensor:     {},
	RolEscribano:      {},
	RolInmobiliaria:   {},
}

// Valido informa si rol es uno de los roles de dominio conocidos.
func (rol Rol) Valido() bool {
	_, ok := rolesValidos[rol]
	return ok
}
