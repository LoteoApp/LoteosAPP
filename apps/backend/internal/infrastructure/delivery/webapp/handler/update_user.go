package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/users"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/users"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type UpdateUserHandler struct {
	updateUser users.UpdateUser
}

func NewUpdateUserHandler(updateUser users.UpdateUser) *UpdateUserHandler {
	return &UpdateUserHandler{updateUser: updateUser}
}

// Handle updates an existing user managed by this ABM for administrador
// callers. It must run behind middleware.RequireAuth.
func (handler *UpdateUserHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())
	id := request.PathValue("id")
	if !isValidUUID(id) {
		return domain.ErrUsuarioIDInvalido
	}

	body, err := decodeJSON[dto.UpdateUserRequest](request)
	if err != nil {
		return err
	}

	usuario, err := handler.updateUser.Execute(request.Context(), users.UpdateUserInput{
		ActorRoles: principal.Roles,
		Subject:    principal.Subject,
		ID:         id,
		Nombre:     body.Nombre,
		Apellido:   body.Apellido,
	})
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.UserResponse{Usuario: usuario})
	return nil
}
