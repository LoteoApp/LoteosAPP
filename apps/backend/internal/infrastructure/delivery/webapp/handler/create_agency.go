package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/agencies"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/agencies"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

// An inmobiliaria is four short text fields, so the cap stops a caller from
// making the decoder allocate before the use case gets to check the role.
const maxAgencyBodyBytes = 32 << 10

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

	request.Body = http.MaxBytesReader(w, request.Body, maxAgencyBodyBytes)

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
