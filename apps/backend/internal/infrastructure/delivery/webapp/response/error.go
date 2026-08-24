package response

import (
	"errors"
	"log/slog"
	"net/http"

	"loteosapp/backend/internal/business/domain"
)

// WriteError maps err to an HTTP response. A *domain.Error is written using
// its own Code, Message and Kind (translated to a status via statusForKind);
// any other error is logged with logMsg and hidden behind a generic 500, so
// internal details never reach the client.
func WriteError(w http.ResponseWriter, request *http.Request, logMsg string, err error) {
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		WriteJSON(w, statusForKind(domainErr.Kind), ErrorResponse{
			Code: domainErr.Code, Message: domainErr.Message,
		})
		return
	}

	slog.ErrorContext(request.Context(), logMsg, "error", err)
	WriteJSON(w, http.StatusInternalServerError, ErrorResponse{
		Code: "internal_error", Message: "Ocurrió un error inesperado",
	})
}

func WriteBadRequest(w http.ResponseWriter, code, message string) {
	WriteJSON(w, http.StatusBadRequest, ErrorResponse{Code: code, Message: message})
}

func statusForKind(kind domain.Kind) int {
	switch kind {
	case domain.KindInvalid:
		return http.StatusBadRequest
	case domain.KindForbidden:
		return http.StatusForbidden
	case domain.KindConflict:
		return http.StatusConflict
	case domain.KindNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
