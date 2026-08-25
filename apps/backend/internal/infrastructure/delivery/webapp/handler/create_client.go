package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/clients"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/clients"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type CreateClientHandler struct {
	createClient clients.CreateClient
}

func NewCreateClientHandler(createClient clients.CreateClient) *CreateClientHandler {
	return &CreateClientHandler{createClient: createClient}
}

// Handle handles cliente sign-up for administrador, administrativo and
// inmobiliaria callers. It must run behind middleware.RequireAuth.
func (handler *CreateClientHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())

	body, err := decodeJSON[dto.CreateClientRequest](request)
	if err != nil {
		return err
	}

	cliente, err := handler.createClient.Execute(
		request.Context(), principal.Roles, principal.Subject,
		body.Nombre, body.Apellido, body.DNI, body.Celular, body.Email,
	)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusCreated, dto.ClientResponse{Cliente: cliente})
	return nil
}
