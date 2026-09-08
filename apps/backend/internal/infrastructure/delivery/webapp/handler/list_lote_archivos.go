package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/loteos"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ListLoteFilesHandler struct {
	listLoteFiles loteos.ListLoteFiles
}

func NewListLoteFilesHandler(listLoteFiles loteos.ListLoteFiles) *ListLoteFilesHandler {
	return &ListLoteFilesHandler{listLoteFiles: listLoteFiles}
}

// Handle lists the active fotos/planos attached to one lote. It must run
// behind middleware.RequireAuth.
func (handler *ListLoteFilesHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	files, err := handler.listLoteFiles.Execute(
		request.Context(), actor, request.PathValue("loteoId"), request.PathValue("loteId"),
	)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.ListFilesResponse{Files: dto.FilesFromDomain(files)})
	return nil
}
