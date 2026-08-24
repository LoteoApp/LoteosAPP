package handler

import (
	"net/http"

	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

// HTTPHandler is a single route's handler. It returns an error instead of
// writing one directly, so every route's failure path goes through the same
// response.WriteError translation.
type HTTPHandler interface {
	Handle(w http.ResponseWriter, r *http.Request) error
}

// Adapt turns an HTTPHandler into a standard http.HandlerFunc.
func Adapt(h HTTPHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h.Handle(w, r); err != nil {
			response.WriteError(w, r, "request failed: "+r.Method+" "+r.URL.Path, err)
		}
	}
}
