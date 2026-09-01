package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/agencies"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/agencies"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type CreateAgencyHandler struct {
	createAgency agencies.CreateAgency
}

func NewCreateAgencyHandler(createAgency agencies.CreateAgency) *CreateAgencyHandler {
	return &CreateAgencyHandler{createAgency: createAgency}
}

// Handle registers a new inmobiliaria for administrador callers. It must run
// behind middleware.RequireAuth.
func (handler *CreateAgencyHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())

	body, err := decodeJSON[dto.CreateAgencyRequest](request)
	if err != nil {
		return err
	}

	inmobiliaria, err := handler.createAgency.Execute(request.Context(), agencies.CreateAgencyInput{
		ActorRoles:  principal.Roles,
		Subject:     principal.Subject,
		RazonSocial: body.RazonSocial,
		CUIT:        body.CUIT,
		Telefono:    body.Telefono,
		Email:       body.Email,
	})
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusCreated, dto.AgencyResponse{Inmobiliaria: inmobiliaria})
	return nil
}
