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

// authorizeCollector reads the caller's usuarios row and derives what they
// may reach from its current rol. The token's roles are not trusted here: a
// token issued before a demotion keeps the old rol until it expires, and an
// ex-administrador must not keep reading or collecting on every venta.
func authorizeCollector(ctx context.Context, users gateway.UserRepository, actor Actor) (domain.Usuario, gateway.SaleScope, error) {
	usuario, err := users.FindByAuthProviderID(ctx, actor.AuthProviderID)
	if err != nil {
		if errors.Is(err, domain.ErrUsuarioNoEncontrado) {
			return domain.Usuario{}, gateway.SaleScope{}, domain.ErrActorNoAprovisionado
		}
		return domain.Usuario{}, gateway.SaleScope{}, fromRepository(err)
	}
	if !usuario.Activo() {
		return domain.Usuario{}, gateway.SaleScope{}, domain.ErrCuentaInactiva
	}
	scope, err := collectionScope(usuario.Rol, actor.AuthProviderID)
	if err != nil {
		return domain.Usuario{}, gateway.SaleScope{}, err
	}
	return usuario, scope, nil
}

// collectionScope mirrors the ventas scope: internal users reach every
// venta, an agency user only those sold by their own agency.
func collectionScope(rol domain.Rol, authProviderID string) (gateway.SaleScope, error) {
	if !domain.IsCollectionRole(rol) {
		return gateway.SaleScope{}, domain.ErrNoAutorizado
	}
	if rol != domain.RolInmobiliaria {
		return gateway.SaleScope{}, nil
	}
	return gateway.SaleScope{AssigneeAuthProviderID: &authProviderID, ByAgency: true}, nil
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
