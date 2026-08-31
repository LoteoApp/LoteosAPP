package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/surveyors"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/surveyors"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type CreateSurveyorHandler struct {
	createSurveyor surveyors.CreateSurveyor
}

func NewCreateSurveyorHandler(createSurveyor surveyors.CreateSurveyor) *CreateSurveyorHandler {
	return &CreateSurveyorHandler{createSurveyor: createSurveyor}
}

// Handle handles the administrador-only agrimensor sign-up. It must run
// behind middleware.RequireAuth.
func (handler *CreateSurveyorHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())

	body, err := decodeJSON[dto.CreateSurveyorRequest](request)
	if err != nil {
		return err
	}

	agrimensor, temporaryPassword, err := handler.createSurveyor.Execute(
		request.Context(), principal.Roles, body.Nombre, body.Apellido, body.Email,
	)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusCreated, dto.CreateSurveyorResponse{
		Usuario:           agrimensor,
		TemporaryPassword: temporaryPassword,
	})
	return nil
}
