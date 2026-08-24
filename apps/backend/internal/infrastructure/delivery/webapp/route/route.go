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
	mux.HandleFunc("GET /readyz", checkDatabaseReadiness.Ready)
	mux.HandleFunc("GET /api/v1/system", getSystemInfo.Info)

	requireAuth := middleware.RequireAuth(verifier)
	mux.Handle("POST /api/v1/usuarios", requireAuth(http.HandlerFunc(createUser.Create)))
	mux.Handle("PATCH /api/v1/usuarios/me", requireAuth(http.HandlerFunc(completeProfile.CompleteProfile)))
}
