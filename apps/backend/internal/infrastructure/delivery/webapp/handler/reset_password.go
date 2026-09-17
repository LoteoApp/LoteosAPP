package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/users"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/users"
)

// The body is a token plus a password, so this cap stops an unauthenticated
// caller from making the decoder allocate on an oversized payload.
const maxResetPasswordBytes = 4 << 10

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
	request.Body = http.MaxBytesReader(w, request.Body, maxResetPasswordBytes)

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
