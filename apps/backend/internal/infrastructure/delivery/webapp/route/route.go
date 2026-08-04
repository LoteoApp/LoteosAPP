package route

import (
	"net/http"

	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
)

func RegisterRoutes(mux *http.ServeMux, h *handler.Handler) {
	mux.HandleFunc("GET /healthz", h.Live)
	mux.HandleFunc("GET /readyz", h.Ready)
	mux.HandleFunc("GET /api/v1/system", h.Info)
}
