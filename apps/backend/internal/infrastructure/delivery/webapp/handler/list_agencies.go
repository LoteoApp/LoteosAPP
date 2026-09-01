package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/agencies"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/agencies"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ListAgenciesHandler struct {
	listAgencies agencies.ListAgencies
}

func NewListAgenciesHandler(listAgencies agencies.ListAgencies) *ListAgenciesHandler {
	return &ListAgenciesHandler{listAgencies: listAgencies}
}

// Handle searches active inmobiliarias by razón social or CUIT via the ?q=
// query param, for administrador and administrativo callers. It must run
// behind middleware.RequireAuth.
func (handler *ListAgenciesHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())
	search := request.URL.Query().Get("q")

	inmobiliarias, err := handler.listAgencies.Execute(request.Context(), agencies.ListAgenciesInput{
		ActorRoles: principal.Roles,
		Search:     search,
	})
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.ListAgenciesResponse{Inmobiliarias: inmobiliarias})
	return nil
}
