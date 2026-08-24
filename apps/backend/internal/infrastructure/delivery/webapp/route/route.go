package route

import (
	"net/http"

	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

func RegisterRoutes(
	mux *http.ServeMux,
	h *handler.Handler,
	uh *handler.UserHandler,
	ch *handler.ClientHandler,
	verifier *supabase.Verifier,
) {
	mux.HandleFunc("GET /healthz", h.Live)
	mux.HandleFunc("GET /readyz", h.Ready)
	mux.HandleFunc("GET /api/v1/system", h.Info)

	requireAuth := middleware.RequireAuth(verifier)
	mux.Handle("POST /api/v1/usuarios", requireAuth(http.HandlerFunc(uh.Create)))
	mux.Handle("PATCH /api/v1/usuarios/me", requireAuth(http.HandlerFunc(uh.CompleteProfile)))

	mux.Handle("GET /api/v1/clientes", requireAuth(http.HandlerFunc(ch.List)))
	mux.Handle("POST /api/v1/clientes", requireAuth(http.HandlerFunc(ch.Create)))
	mux.Handle("PATCH /api/v1/clientes/{id}", requireAuth(http.HandlerFunc(ch.Update)))
	mux.Handle("DELETE /api/v1/clientes/{id}", requireAuth(http.HandlerFunc(ch.Delete)))
}
