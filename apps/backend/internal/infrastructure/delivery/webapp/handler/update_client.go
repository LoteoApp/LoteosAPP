package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/clients"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/clients"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type UpdateClientHandler struct {
	updateClient clients.UpdateClient
}

func NewUpdateClientHandler(updateClient clients.UpdateClient) *UpdateClientHandler {
	return &UpdateClientHandler{updateClient: updateClient}
}

// Handle updates an existing cliente for administrador, administrativo and
// inmobiliaria callers. It must run behind middleware.RequireAuth.
func (handler *UpdateClientHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())
	id := request.PathValue("id")

	body, err := decodeJSON[dto.UpdateClientRequest](request)
	if err != nil {
		return err
	}

	cliente, err := handler.updateClient.Execute(
		request.Context(), principal.Roles, principal.Subject, id,
		body.Nombre, body.Apellido, body.DNI, body.Celular, body.Email,
	)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.ClientResponse{Cliente: cliente})
	return nil
}
