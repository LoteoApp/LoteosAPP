package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/surveyors"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/surveyors"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ListSurveyorsHandler struct {
	listSurveyors surveyors.ListSurveyors
}

func NewListSurveyorsHandler(listSurveyors surveyors.ListSurveyors) *ListSurveyorsHandler {
	return &ListSurveyorsHandler{listSurveyors: listSurveyors}
}

// Handle lists the agrimensores for administrador callers. Only the active
// ones unless the request asks for the rest with ?incluirBajas=true. It must
// run behind middleware.RequireAuth.
func (handler *ListSurveyorsHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())
	includeInactive := request.URL.Query().Get("incluirBajas") == "true"

	agrimensores, err := handler.listSurveyors.Execute(request.Context(), principal.Roles, includeInactive)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.ListSurveyorsResponse{Agrimensores: agrimensores})
	return nil
}
