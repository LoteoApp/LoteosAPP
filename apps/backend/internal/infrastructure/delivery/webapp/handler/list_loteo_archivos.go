package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/loteos"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ListLoteoArchivosHandler struct {
	listLoteoArchivos loteos.ListLoteoArchivos
}

func NewListLoteoArchivosHandler(listLoteoArchivos loteos.ListLoteoArchivos) *ListLoteoArchivosHandler {
	return &ListLoteoArchivosHandler{listLoteoArchivos: listLoteoArchivos}
}

// Handle lists the active fotos/planos attached to a loteo. It must run
// behind middleware.RequireAuth.
func (handler *ListLoteoArchivosHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	archivos, err := handler.listLoteoArchivos.Execute(request.Context(), actor, request.PathValue("loteoId"))
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.ListArchivosResponse{Archivos: dto.ArchivosFromDomain(archivos)})
	return nil
}
