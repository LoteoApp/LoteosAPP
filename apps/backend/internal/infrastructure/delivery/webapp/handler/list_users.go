package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/users"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/users"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ListUsersHandler struct {
	listUsers users.ListUsers
}

func NewListUsersHandler(listUsers users.ListUsers) *ListUsersHandler {
	return &ListUsersHandler{listUsers: listUsers}
}

// Handle lists the users this ABM manages (administrativo, escribano,
// inmobiliaria, agrimensor) for administrador callers. Only the active
// ones unless the request asks for the rest with ?incluirBajas=true. It
// must run behind middleware.RequireAuth.
func (handler *ListUsersHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())
	includeInactive := request.URL.Query().Get("incluirBajas") == "true"

	usuarios, err := handler.listUsers.Execute(request.Context(), principal.Roles, includeInactive)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.ListUsersResponse{Usuarios: usuarios})
	return nil
}
