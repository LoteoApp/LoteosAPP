package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/users"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/users"
)

type ResetPasswordHandler struct {
	resetPassword users.ResetPassword
}

func NewResetPasswordHandler(resetPassword users.ResetPassword) *ResetPasswordHandler {
	return &ResetPasswordHandler{resetPassword: resetPassword}
}

// Handle confirms a password reset with the token from the email
// RequestPasswordReset sent. Unauthenticated, like that request: proving
// control of the token stands in for a session here.
func (handler *ResetPasswordHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	body, err := decodeJSON[dto.ResetPasswordRequest](request)
	if err != nil {
		return err
	}

	if err := handler.resetPassword.Execute(request.Context(), body.Token, body.NewPassword); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
