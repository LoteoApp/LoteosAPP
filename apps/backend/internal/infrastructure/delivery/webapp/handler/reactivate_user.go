package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/users"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/users"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ReactivateUserHandler struct {
	reactivateUser users.ReactivateUser
}

func NewReactivateUserHandler(reactivateUser users.ReactivateUser) *ReactivateUserHandler {
	return &ReactivateUserHandler{reactivateUser: reactivateUser}
}

// Handle undoes a baja on a user managed by this ABM. Only administrador
// callers may do this. It must run behind middleware.RequireAuth.
func (handler *ReactivateUserHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())
	id := request.PathValue("id")
	if !isValidUUID(id) {
		return domain.ErrUsuarioIDInvalido
	}

	usuario, err := handler.reactivateUser.Execute(request.Context(), principal.Roles, principal.Subject, id)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.UserResponse{Usuario: usuario})
	return nil
}
