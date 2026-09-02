package users

import "loteosapp/backend/internal/business/domain"

// gestionableRoles are the roles this ABM manages. Agrimensor has its own
// module (features/surveyors, /api/v1/agrimensores) and administrador
// accounts aren't managed through here, so both stay out of reach: a
// request naming either is treated the same as a request naming a user
// that doesn't exist.
var gestionableRoles = []domain.Rol{
	domain.RolAdministrativo,
	domain.RolEscribano,
	domain.RolInmobiliaria,
}

func esRolGestionable(rol domain.Rol) bool {
	for _, candidate := range gestionableRoles {
		if candidate == rol {
			return true
		}
	}

	return false
}
