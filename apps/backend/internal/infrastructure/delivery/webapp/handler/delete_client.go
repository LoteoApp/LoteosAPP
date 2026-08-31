package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/clients"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type DeleteClientHandler struct {
	deleteClient clients.DeleteClient
}

func NewDeleteClientHandler(deleteClient clients.DeleteClient) *DeleteClientHandler {
	return &DeleteClientHandler{deleteClient: deleteClient}
}

// Handle gives a cliente de baja. Only administrador callers may do this. It
// must run behind middleware.RequireAuth.
func (handler *DeleteClientHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())
	id := request.PathValue("id")
	if !isValidUUID(id) {
		return domain.ErrClienteIDInvalido
	}

	input := clients.DeleteClientInput{ActorRoles: principal.Roles, Subject: principal.Subject, ID: id}
	if err := handler.deleteClient.Execute(request.Context(), input); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
