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

type CompleteProfileHandler struct {
	completeProfile users.CompleteProfile
}

func NewCompleteProfileHandler(completeProfile users.CompleteProfile) *CompleteProfileHandler {
	return &CompleteProfileHandler{completeProfile: completeProfile}
}

// CompleteProfile lets the authenticated caller fill in their own profile.
// It must run behind middleware.RequireAuth.
func (handler *CompleteProfileHandler) CompleteProfile(w http.ResponseWriter, request *http.Request) {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())

	body, ok := decodeJSON[dto.CompleteProfileRequest](w, request)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()

	usuario, err := handler.completeProfile.Execute(ctx, principal.Subject, body.Nombre, body.Apellido)
	if err != nil {
		response.WriteError(w, request, "complete profile failed", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, usuario)
}
