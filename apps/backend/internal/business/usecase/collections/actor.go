package collections

import (
	"context"
	"errors"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type Actor struct {
	AuthProviderID string
	Roles          []string
}

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

func clockOrSystem(clocks []Clock) Clock {
	if len(clocks) > 0 && clocks[0] != nil {
		return clocks[0]
	}
	return SystemClock{}
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

// collectionScope mirrors the ventas scope: internal users reach every
// venta, an agency user only those sold by their own agency.
func collectionScope(actor Actor) (gateway.SaleScope, error) {
	if domain.HasRole(actor.Roles, domain.RolAdministrador) || domain.HasRole(actor.Roles, domain.RolAdministrativo) {
		return gateway.SaleScope{}, nil
	}
	if !domain.HasRole(actor.Roles, domain.RolInmobiliaria) {
		return gateway.SaleScope{}, domain.ErrNoAutorizado
	}
	id := actor.AuthProviderID
	return gateway.SaleScope{AssigneeAuthProviderID: &id, ByAgency: true}, nil
}

func hasCollectionRole(roles []string) bool {
	for _, role := range roles {
		if domain.IsCollectionRole(domain.Rol(role)) {
			return true
		}
	}
	return false
}

func fromRepository(err error) error {
	if err == nil {
		return nil
	}
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		return err
	}
	return domain.ErrDatabaseUnavailable.WithCause(err)
}
