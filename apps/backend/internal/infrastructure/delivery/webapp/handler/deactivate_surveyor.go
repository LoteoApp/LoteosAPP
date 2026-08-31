package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/surveyors"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type DeactivateSurveyorHandler struct {
	deactivateSurveyor surveyors.DeactivateSurveyor
}

func NewDeactivateSurveyorHandler(deactivateSurveyor surveyors.DeactivateSurveyor) *DeactivateSurveyorHandler {
	return &DeactivateSurveyorHandler{deactivateSurveyor: deactivateSurveyor}
}

// Handle gives an agrimensor de baja. Only administrador callers may do this.
// It must run behind middleware.RequireAuth.
func (handler *DeactivateSurveyorHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())
	id := request.PathValue("id")
	if !isValidUUID(id) {
		return domain.ErrAgrimensorIDInvalido
	}

	if err := handler.deactivateSurveyor.Execute(request.Context(), principal.Roles, principal.Subject, id); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
