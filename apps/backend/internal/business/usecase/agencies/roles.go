package agencies

import "loteosapp/backend/internal/business/domain"

// hasRole reports whether actorRoles holds any of allowed.
func hasRole(actorRoles []string, allowed ...string) bool {
	for _, role := range allowed {
		if domain.HasRole(actorRoles, role) {
			return true
		}
	}

	return false
}
