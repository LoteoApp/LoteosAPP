package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/loteos"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ListLoteoFilesHandler struct {
	listLoteoFiles loteos.ListLoteoFiles
}

func NewListLoteoFilesHandler(listLoteoFiles loteos.ListLoteoFiles) *ListLoteoFilesHandler {
	return &ListLoteoFilesHandler{listLoteoFiles: listLoteoFiles}
}

// Handle lists the active fotos/planos attached to a loteo. It must run
// behind middleware.RequireAuth.
func (handler *ListLoteoFilesHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	files, err := handler.listLoteoFiles.Execute(request.Context(), actor, request.PathValue("loteoId"))
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.ListFilesResponse{Files: dto.FilesFromDomain(files)})
	return nil
}
