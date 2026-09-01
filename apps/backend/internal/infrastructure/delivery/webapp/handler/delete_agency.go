package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/agencies"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type DeleteAgencyHandler struct {
	deleteAgency agencies.DeleteAgency
}

func NewDeleteAgencyHandler(deleteAgency agencies.DeleteAgency) *DeleteAgencyHandler {
	return &DeleteAgencyHandler{deleteAgency: deleteAgency}
}

// Handle gives an inmobiliaria de baja. Only administrador callers may do
// this. It must run behind middleware.RequireAuth.
func (handler *DeleteAgencyHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())
	id := request.PathValue("id")
	if !isValidUUID(id) {
		return domain.ErrInmobiliariaIDInvalido
	}

	input := agencies.DeleteAgencyInput{ActorRoles: principal.Roles, Subject: principal.Subject, ID: id}
	if err := handler.deleteAgency.Execute(request.Context(), input); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
