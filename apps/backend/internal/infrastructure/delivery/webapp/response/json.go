package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON writes payload as the response body. A failure here can't change
// the status already sent, but it means the client never saw the result of
// work that may have been committed, so it's logged instead of dropped.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("writing the response body failed", "status", status, "error", err)
	}
}
