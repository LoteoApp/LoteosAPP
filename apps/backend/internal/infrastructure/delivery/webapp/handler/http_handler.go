package handler

import (
	"context"
	"net/http"
	"time"

	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

// HTTPHandler is a single route's handler. It returns an error instead of
// writing one directly, so every route's failure path goes through the same
// response.WriteError translation.
type HTTPHandler interface {
	Handle(w http.ResponseWriter, r *http.Request) error
}

// Adapt turns an HTTPHandler into a standard http.HandlerFunc, bounding the
// request's context to timeout so individual handlers don't each set up
// their own context.WithTimeout.
func Adapt(h HTTPHandler, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		if err := h.Handle(w, r.WithContext(ctx)); err != nil {
			response.WriteError(w, r, "request failed: "+r.Method+" "+r.URL.Path, err)
		}
	}
}
