package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

const maxUpdateCalleBytes = 8 << 10

type UpdateCalleHandler struct {
	updateCalle loteos.UpdateCalle
}

func NewUpdateCalleHandler(updateCalle loteos.UpdateCalle) *UpdateCalleHandler {
	return &UpdateCalleHandler{updateCalle: updateCalle}
}

func (handler *UpdateCalleHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())

	request.Body = http.MaxBytesReader(w, request.Body, maxUpdateCalleBytes)

	body, err := decodeJSON[dto.UpdateCalleRequest](request)
	if err != nil {
		return err
	}

	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	calle, err := handler.updateCalle.Execute(
		request.Context(),
		actor,
		request.PathValue("loteoId"),
		request.PathValue("calleId"),
		domain.CalleData{Name: body.Name, Type: body.Type},
	)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, calle)
	return nil
}
