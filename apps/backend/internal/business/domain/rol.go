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

// HasRole reports whether an actor carrying roles holds role. Supabase sends
// a single role per user today, but the claim is read as a list so an extra
// role in the token doesn't change how a permission is checked.
func HasRole(roles []string, role string) bool {
	for _, candidate := range roles {
		if candidate == role {
			return true
		}
	}

	return false
}
