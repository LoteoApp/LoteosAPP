package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/surveyors"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/surveyors"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type UpdateSurveyorHandler struct {
	updateSurveyor surveyors.UpdateSurveyor
}

func NewUpdateSurveyorHandler(updateSurveyor surveyors.UpdateSurveyor) *UpdateSurveyorHandler {
	return &UpdateSurveyorHandler{updateSurveyor: updateSurveyor}
}

// Handle updates an existing agrimensor for administrador callers. It must
// run behind middleware.RequireAuth.
func (handler *UpdateSurveyorHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())
	id := request.PathValue("id")
	if !isValidUUID(id) {
		return domain.ErrAgrimensorIDInvalido
	}

	body, err := decodeJSON[dto.UpdateSurveyorRequest](request)
	if err != nil {
		return err
	}

	agrimensor, err := handler.updateSurveyor.Execute(
		request.Context(), principal.Roles, principal.Subject, id, body.Nombre, body.Apellido,
	)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.SurveyorResponse{Usuario: agrimensor})
	return nil
}
