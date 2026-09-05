package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/loteos"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ListLoteArchivosHandler struct {
	listLoteArchivos loteos.ListLoteArchivos
}

func NewListLoteArchivosHandler(listLoteArchivos loteos.ListLoteArchivos) *ListLoteArchivosHandler {
	return &ListLoteArchivosHandler{listLoteArchivos: listLoteArchivos}
}

// Handle lists the active fotos/planos attached to one lote. It must run
// behind middleware.RequireAuth.
func (handler *ListLoteArchivosHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	archivos, err := handler.listLoteArchivos.Execute(
		request.Context(), actor, request.PathValue("loteoId"), request.PathValue("loteId"),
	)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.ListArchivosResponse{Archivos: dto.ArchivosFromDomain(archivos)})
	return nil
}
