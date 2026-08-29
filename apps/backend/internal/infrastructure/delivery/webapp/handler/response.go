package handler

import (
	"encoding/json"
	"net/http"

	"loteosapp/backend/internal/business/domain"
)

// decodeJSON decodes the request body into T, returning a *domain.Error
// (KindInvalid) if the body isn't valid JSON. The decoder's error travels as
// Cause so it reaches the log without being shown to the caller: it's the
// only way to tell malformed JSON from a body that hit a size limit.
func decodeJSON[T any](request *http.Request) (T, error) {
	var body T
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		return body, &domain.Error{
			Kind:    domain.KindInvalid,
			Code:    "invalid_body",
			Message: "Cuerpo de la solicitud inválido",
			Cause:   err,
		}
	}
	return body, nil
}
