package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/loteos"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ListLoteosHandler struct {
	listLoteos loteos.ListLoteos
}

func NewListLoteosHandler(listLoteos loteos.ListLoteos) *ListLoteosHandler {
	return &ListLoteosHandler{listLoteos: listLoteos}
}

// Handle lists the loteos the caller may see, filtered by the ?q= query
// param. It must run behind middleware.RequireAuth.
func (handler *ListLoteosHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())

	result, err := handler.listLoteos.Execute(request.Context(), loteos.ListLoteosInput{
		Actor:  loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles},
		Search: request.URL.Query().Get("q"),
	})
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.ListLoteosResponse{Loteos: result})
	return nil
}
