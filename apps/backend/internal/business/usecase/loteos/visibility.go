package loteos

import (
	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// loteoVisibility resolves which loteos an actor may read. administrador and
// administrativo see every loteo (an unrestricted scope). agrimensor and
// escribano see the loteos assigned to them through usuario_loteos;
// inmobiliaria sees the ones reached through its agency (inmobiliaria_loteos).
// Each role enables only its own assignment path, so an agrimensor tied to an
// agency by mistake can't read its loteos and an inmobiliaria user can't read
// a loteo through a stray direct assignment. Any other actor gets
// domain.ErrNoAutorizado: read access is denied by default.
func loteoVisibility(actor Actor) (gateway.LoteoScope, error) {
	if domain.HasRole(actor.Roles, domain.RolAdministrador) ||
		domain.HasRole(actor.Roles, domain.RolAdministrativo) {
		return gateway.LoteoScope{}, nil
	}

	scope := gateway.LoteoScope{
		ByUserAssignment: domain.HasRole(actor.Roles, domain.RolAgrimensor) ||
			domain.HasRole(actor.Roles, domain.RolEscribano),
		ByAgencyAssignment: domain.HasRole(actor.Roles, domain.RolInmobiliaria),
	}
	if !scope.ByUserAssignment && !scope.ByAgencyAssignment {
		return gateway.LoteoScope{}, domain.ErrNoAutorizado
	}

	id := actor.AuthProviderID
	scope.AssigneeAuthProviderID = &id

	return scope, nil
}
