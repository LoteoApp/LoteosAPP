package reservations

import (
	"context"
	"errors"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type Actor struct {
	AuthProviderID string
	Roles          []string
}

func resolveActor(ctx context.Context, users gateway.UserRepository, actor Actor) (domain.Usuario, error) {
	usuario, err := users.FindByAuthProviderID(ctx, actor.AuthProviderID)
	if err != nil {
		if errors.Is(err, domain.ErrUsuarioNoEncontrado) {
			return domain.Usuario{}, domain.ErrActorNoAprovisionado
		}
		return domain.Usuario{}, err
	}
	if !usuario.Activo() {
		return domain.Usuario{}, domain.ErrCuentaInactiva
	}
	return usuario, nil
}

func reservationScope(actor Actor) (gateway.ReservationScope, error) {
	if domain.HasRole(actor.Roles, domain.RolAdministrador) || domain.HasRole(actor.Roles, domain.RolAdministrativo) {
		return gateway.ReservationScope{}, nil
	}
	if !domain.HasRole(actor.Roles, domain.RolInmobiliaria) {
		return gateway.ReservationScope{}, domain.ErrNoAutorizado
	}
	id := actor.AuthProviderID
	return gateway.ReservationScope{AssigneeAuthProviderID: &id, ByAgencyAssignment: true}, nil
}

func hasReservationWriteRole(roles []string) bool {
	for _, role := range roles {
		if domain.IsReservationRole(domain.Rol(role)) {
			return true
		}
	}
	return false
}
