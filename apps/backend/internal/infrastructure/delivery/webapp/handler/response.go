package handler

import (
	"encoding/json"
	"net/http"

	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

// decodeJSON decodes the request body into T, writing a 400 response and
// returning ok=false if the body isn't valid JSON.
func decodeJSON[T any](w http.ResponseWriter, request *http.Request) (body T, ok bool) {
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		response.WriteBadRequest(w, "invalid_body", "Cuerpo de la solicitud inválido")
		return body, false
	}
	return body, true
}
