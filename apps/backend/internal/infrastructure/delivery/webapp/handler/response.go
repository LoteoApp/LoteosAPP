package handler

import (
	"encoding/json"
	"net/http"

	"loteosapp/backend/internal/business/domain"
)

// decodeJSON decodes the request body into T, returning a *domain.Error
// (KindInvalid) if the body isn't valid JSON.
func decodeJSON[T any](request *http.Request) (T, error) {
	var body T
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		return body, &domain.Error{
			Kind:    domain.KindInvalid,
			Code:    "invalid_body",
			Message: "Cuerpo de la solicitud inválido",
		}
	}
	return body, nil
}
