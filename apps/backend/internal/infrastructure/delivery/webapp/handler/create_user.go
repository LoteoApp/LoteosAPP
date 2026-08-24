package handler

import (
	"context"
	"net/http"
	"time"

	"loteosapp/backend/internal/business/usecase/users"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/users"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type CreateUserHandler struct {
	createUser users.CreateUser
}

func NewCreateUserHandler(createUser users.CreateUser) *CreateUserHandler {
	return &CreateUserHandler{createUser: createUser}
}

// Create handles the administrador-only user sign-up. It must run behind
// middleware.RequireAuth.
func (handler *CreateUserHandler) Create(w http.ResponseWriter, request *http.Request) {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())

	body, ok := decodeJSON[dto.CreateUserRequest](w, request)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()

	usuario, temporaryPassword, err := handler.createUser.Execute(ctx, principal.Roles, body.Email, body.Rol)
	if err != nil {
		writeUserError(w, request, "create user failed", err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, dto.CreateUserResponse{
		Usuario:           usuario,
		TemporaryPassword: temporaryPassword,
	})
}
