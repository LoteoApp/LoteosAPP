package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type DeleteArchivoHandler struct {
	deleteArchivo loteos.DeleteArchivo
}

func NewDeleteArchivoHandler(deleteArchivo loteos.DeleteArchivo) *DeleteArchivoHandler {
	return &DeleteArchivoHandler{deleteArchivo: deleteArchivo}
}

// Handle soft-deletes one foto/plano. It must run behind
// middleware.RequireAuth.
func (handler *DeleteArchivoHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	err := handler.deleteArchivo.Execute(
		request.Context(), actor, request.PathValue("loteoId"), request.PathValue("archivoId"),
	)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
