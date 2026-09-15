package sales

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

func saleScope(actor Actor) (gateway.SaleScope, error) {
	if domain.HasRole(actor.Roles, domain.RolAdministrador) || domain.HasRole(actor.Roles, domain.RolAdministrativo) {
		return gateway.SaleScope{}, nil
	}
	if !domain.HasRole(actor.Roles, domain.RolInmobiliaria) {
		return gateway.SaleScope{}, domain.ErrNoAutorizado
	}
	id := actor.AuthProviderID
	return gateway.SaleScope{AssigneeAuthProviderID: &id, ByAgency: true}, nil
}

func hasSaleRole(roles []string) bool {
	for _, role := range roles {
		if domain.IsSaleRole(domain.Rol(role)) {
			return true
		}
	}
	return false
}
