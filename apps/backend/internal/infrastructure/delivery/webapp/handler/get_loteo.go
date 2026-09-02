package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type GetLoteoHandler struct {
	getLoteo loteos.GetLoteo
}

func NewGetLoteoHandler(getLoteo loteos.GetLoteo) *GetLoteoHandler {
	return &GetLoteoHandler{getLoteo: getLoteo}
}

// Handle returns one loteo with its plan and geometry. A loteo the caller
// may not see reads as a 404, not a 403. It must run behind
// middleware.RequireAuth.
func (handler *GetLoteoHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())

	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	loteo, err := handler.getLoteo.Execute(request.Context(), actor, request.PathValue("loteoId"))
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, loteo)
	return nil
}
