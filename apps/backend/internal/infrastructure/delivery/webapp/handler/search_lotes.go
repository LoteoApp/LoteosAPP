package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type SearchLotesHandler struct {
	searchLotes loteos.SearchLotes
}

func NewSearchLotesHandler(searchLotes loteos.SearchLotes) *SearchLotesHandler {
	return &SearchLotesHandler{searchLotes: searchLotes}
}

// Handle lists the lotes the caller may sell, filtered by the ?q= and
// ?estado= query params. It must run behind middleware.RequireAuth.
func (handler *SearchLotesHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())

	result, err := handler.searchLotes.Execute(request.Context(), loteos.SearchLotesInput{
		Actor:  loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles},
		Search: request.URL.Query().Get("q"),
		State:  domain.LotState(request.URL.Query().Get("estado")),
	})
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.SearchLotesResponse{Lotes: result})
	return nil
}
