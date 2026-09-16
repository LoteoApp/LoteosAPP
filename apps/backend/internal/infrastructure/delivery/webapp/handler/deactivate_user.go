package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/users"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type DeactivateUserHandler struct {
	deactivateUser users.DeactivateUser
}

func NewDeactivateUserHandler(deactivateUser users.DeactivateUser) *DeactivateUserHandler {
	return &DeactivateUserHandler{deactivateUser: deactivateUser}
}

// Handle gives a user managed by this ABM de baja. Only administrador
// callers may do this. It must run behind middleware.RequireAuth.
func (handler *DeactivateUserHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())
	id := request.PathValue("id")
	if !isValidUUID(id) {
		return domain.ErrUsuarioIDInvalido
	}

	if err := handler.deactivateUser.Execute(request.Context(), principal.Roles, principal.Subject, id); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
