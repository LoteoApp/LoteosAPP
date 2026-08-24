package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

func writeUserError(w http.ResponseWriter, request *http.Request, logMsg string, err error) {
	switch {
	case errors.Is(err, domain.ErrNoAutorizado):
		response.WriteJSON(w, http.StatusForbidden, response.ErrorResponse{
			Code: "forbidden", Message: "No tenés permisos para esta acción",
		})
	case errors.Is(err, domain.ErrEmailInvalido):
		writeBadRequest(w, "invalid_email", "Email inválido")
	case errors.Is(err, domain.ErrRolInvalido):
		writeBadRequest(w, "invalid_rol", "Rol inválido")
	case errors.Is(err, domain.ErrPerfilInvalido):
		writeBadRequest(w, "invalid_profile", "Nombre y apellido son obligatorios")
	case errors.Is(err, domain.ErrEmailEnUso):
		response.WriteJSON(w, http.StatusConflict, response.ErrorResponse{
			Code: "email_in_use", Message: "El email ya está en uso",
		})
	case errors.Is(err, domain.ErrUsuarioNoEncontrado):
		response.WriteJSON(w, http.StatusNotFound, response.ErrorResponse{
			Code: "user_not_found", Message: "Usuario no encontrado",
		})
	default:
		slog.ErrorContext(request.Context(), logMsg, "error", err)
		response.WriteJSON(w, http.StatusInternalServerError, response.ErrorResponse{
			Code: "internal_error", Message: "Ocurrió un error inesperado",
		})
	}
}
