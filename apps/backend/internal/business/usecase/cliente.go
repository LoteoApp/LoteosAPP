package usecase

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type ClientService struct {
	clientes gateway.ClientRepository
	usuarios gateway.UserRepository
}

func NewClientService(clientes gateway.ClientRepository, usuarios gateway.UserRepository) *ClientService {
	return &ClientService{clientes: clientes, usuarios: usuarios}
}

// List returns the clients matching buscar (name, surname or DNI). Callers
// without a role allowed to manage clients get ErrNoAutorizado.
func (service *ClientService) List(ctx context.Context, actorRoles []string, buscar string) ([]domain.Cliente, error) {
	if !canManageClients(actorRoles) {
		return nil, domain.ErrNoAutorizado
	}

	return service.clientes.List(ctx, strings.TrimSpace(buscar))
}

func (service *ClientService) Create(
	ctx context.Context,
	actorRoles []string,
	actorSubject string,
	cliente domain.Cliente,
) (domain.Cliente, error) {
	if !canManageClients(actorRoles) {
		return domain.Cliente{}, domain.ErrNoAutorizado
	}

	sanitized, err := sanitizeCliente(cliente)
	if err != nil {
		return domain.Cliente{}, err
	}

	actorID, err := service.actorID(ctx, actorSubject)
	if err != nil {
		return domain.Cliente{}, err
	}

	return service.clientes.Create(ctx, sanitized, actorID)
}

func (service *ClientService) Update(
	ctx context.Context,
	actorRoles []string,
	actorSubject string,
	cliente domain.Cliente,
) (domain.Cliente, error) {
	if !canManageClients(actorRoles) {
		return domain.Cliente{}, domain.ErrNoAutorizado
	}

	if strings.TrimSpace(cliente.ID) == "" {
		return domain.Cliente{}, domain.ErrClienteNoEncontrado
	}

	sanitized, err := sanitizeCliente(cliente)
	if err != nil {
		return domain.Cliente{}, err
	}

	actorID, err := service.actorID(ctx, actorSubject)
	if err != nil {
		return domain.Cliente{}, err
	}

	return service.clientes.Update(ctx, sanitized, actorID)
}

// Delete gives a client the baja. Only administrador may do this
// (docs/domain.md, sección "Clientes").
func (service *ClientService) Delete(ctx context.Context, actorRoles []string, actorSubject, id string) error {
	if !hasRole(actorRoles, domain.RolAdministrador) {
		return domain.ErrNoAutorizado
	}

	if strings.TrimSpace(id) == "" {
		return domain.ErrClienteNoEncontrado
	}

	actorID, err := service.actorID(ctx, actorSubject)
	if err != nil {
		return err
	}

	return service.clientes.SoftDelete(ctx, id, actorID)
}

// actorID maps the identity provider subject to the internal user id stored
// in clientes.usuario_modificacion.
func (service *ClientService) actorID(ctx context.Context, actorSubject string) (string, error) {
	usuario, err := service.usuarios.FindByAuthProviderID(ctx, actorSubject)
	if err != nil {
		return "", err
	}

	return usuario.ID, nil
}

func sanitizeCliente(cliente domain.Cliente) (domain.Cliente, error) {
	cliente.Nombre = strings.TrimSpace(cliente.Nombre)
	cliente.Apellido = strings.TrimSpace(cliente.Apellido)
	cliente.DNI = strings.TrimSpace(cliente.DNI)
	cliente.Celular = strings.TrimSpace(cliente.Celular)
	cliente.Email = strings.TrimSpace(cliente.Email)

	if cliente.Nombre == "" || cliente.Apellido == "" || cliente.DNI == "" {
		return domain.Cliente{}, domain.ErrClienteInvalido
	}

	return cliente, nil
}

func canManageClients(roles []string) bool {
	for _, rol := range roles {
		if domain.PuedeGestionarClientes(rol) {
			return true
		}
	}
	return false
}
