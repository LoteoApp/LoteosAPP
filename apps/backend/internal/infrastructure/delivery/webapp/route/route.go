package route

import (
	"net/http"
	"time"

	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

const (
	systemTimeout = 2 * time.Second
	usersTimeout  = 5 * time.Second
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
	mux.HandleFunc("GET /readyz", handler.Adapt(checkDatabaseReadiness, systemTimeout))
	mux.HandleFunc("GET /api/v1/system", handler.Adapt(getSystemInfo, systemTimeout))

	requireAuth := middleware.RequireAuth(verifier)
	mux.Handle("POST /api/v1/usuarios", requireAuth(handler.Adapt(createUser, usersTimeout)))
	mux.Handle("PATCH /api/v1/usuarios/me", requireAuth(handler.Adapt(completeProfile, usersTimeout)))
}
