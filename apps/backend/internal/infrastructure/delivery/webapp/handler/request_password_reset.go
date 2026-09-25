package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/users"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/users"
)

// The body is a single email address, so this cap stops an unauthenticated
// caller from making the decoder allocate on an oversized payload.
const maxRequestPasswordResetBytes = 4 << 10

type RequestPasswordResetHandler struct {
	requestPasswordReset users.RequestPasswordReset
}

func NewRequestPasswordResetHandler(requestPasswordReset users.RequestPasswordReset) *RequestPasswordResetHandler {
	return &RequestPasswordResetHandler{requestPasswordReset: requestPasswordReset}
}

// Handle starts a password reset. It's unauthenticated by nature — the
// caller doesn't have a session yet — and never runs behind
// middleware.RequireAuth.
func (handler *RequestPasswordResetHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	request.Body = http.MaxBytesReader(w, request.Body, maxRequestPasswordResetBytes)

	body, err := decodeJSON[dto.RequestPasswordResetRequest](request)
	if err != nil {
		return err
	}

	if err := handler.requestPasswordReset.Execute(request.Context(), body.Email); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
