package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/agencies"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/agencies"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type UpdateAgencyHandler struct {
	updateAgency agencies.UpdateAgency
}

func NewUpdateAgencyHandler(updateAgency agencies.UpdateAgency) *UpdateAgencyHandler {
	return &UpdateAgencyHandler{updateAgency: updateAgency}
}

// Handle updates an existing inmobiliaria for administrador callers. It must
// run behind middleware.RequireAuth.
func (handler *UpdateAgencyHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())
	id := request.PathValue("id")
	if !isValidUUID(id) {
		return domain.ErrInvalidAgencyID
	}

	request.Body = http.MaxBytesReader(w, request.Body, maxAgencyBodyBytes)

	body, err := decodeJSON[dto.UpdateAgencyRequest](request)
	if err != nil {
		return err
	}

	agency, err := handler.updateAgency.Execute(request.Context(), agencies.UpdateAgencyInput{
		ActorRoles:   principal.Roles,
		Subject:      principal.Subject,
		ID:           id,
		BusinessName: body.BusinessName,
		CUIT:         body.CUIT,
		Phone:        body.Phone,
		Email:        body.Email,
	})
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.AgencyResponse{Agency: agency})
	return nil
}
