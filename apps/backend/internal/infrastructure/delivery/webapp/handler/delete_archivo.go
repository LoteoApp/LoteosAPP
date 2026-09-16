package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type DeleteFileHandler struct {
	deleteFile loteos.DeleteFile
}

func NewDeleteFileHandler(deleteFile loteos.DeleteFile) *DeleteFileHandler {
	return &DeleteFileHandler{deleteFile: deleteFile}
}

// Handle soft-deletes one foto/plano. It must run behind
// middleware.RequireAuth.
func (handler *DeleteFileHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	err := handler.deleteFile.Execute(
		request.Context(), actor, request.PathValue("loteoId"), request.PathValue("archivoId"),
	)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
