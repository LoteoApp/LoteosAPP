package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/clients"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/clients"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ListClientsHandler struct {
	listClients clients.ListClients
}

func NewListClientsHandler(listClients clients.ListClients) *ListClientsHandler {
	return &ListClientsHandler{listClients: listClients}
}

// Handle searches active clientes by nombre, apellido or dni via the ?q=
// query param, for administrador, administrativo and inmobiliaria callers.
// It must run behind middleware.RequireAuth.
func (handler *ListClientsHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())
	search := request.URL.Query().Get("q")

	clientes, err := handler.listClients.Execute(request.Context(), principal.Roles, search)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.ListClientsResponse{Clientes: clientes})
	return nil
}
