package route

import (
	"net/http"

	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

func RegisterRoutes(
	mux *http.ServeMux,
	getSystemInfo *handler.GetSystemInfoHandler,
	checkDatabaseReadiness *handler.CheckDatabaseReadinessHandler,
	createUser *handler.CreateUserHandler,
	completeProfile *handler.CompleteProfileHandler,
	verifier *supabase.Verifier,
) {
	mux.HandleFunc("GET /healthz", handler.Live)
	mux.HandleFunc("GET /readyz", handler.Adapt(checkDatabaseReadiness))
	mux.HandleFunc("GET /api/v1/system", handler.Adapt(getSystemInfo))

	requireAuth := middleware.RequireAuth(verifier)
	mux.Handle("POST /api/v1/usuarios", requireAuth(handler.Adapt(createUser)))
	mux.Handle("PATCH /api/v1/usuarios/me", requireAuth(handler.Adapt(completeProfile)))
}
