package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/users"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type ResendInviteEmailHandler struct {
	resendInviteEmail users.ResendInviteEmail
}

func NewResendInviteEmailHandler(resendInviteEmail users.ResendInviteEmail) *ResendInviteEmailHandler {
	return &ResendInviteEmailHandler{resendInviteEmail: resendInviteEmail}
}

// Handle resends a usuario's invite email with a fresh temporary password,
// for when CreateUser's own best-effort send failed. Only administrador
// callers may do this. It must run behind middleware.RequireAuth.
func (handler *ResendInviteEmailHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	id := request.PathValue("id")
	if !isValidUUID(id) {
		return domain.ErrUsuarioIDInvalido
	}

	if err := handler.resendInviteEmail.Execute(request.Context(), principal.Roles, id); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
