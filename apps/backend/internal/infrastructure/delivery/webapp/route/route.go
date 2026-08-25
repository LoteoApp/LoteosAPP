package route

import (
	"net/http"
	"time"

	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

const usersTimeout = 5 * time.Second

func RegisterRoutes(
	mux *http.ServeMux,
	createUser *handler.CreateUserHandler,
	completeProfile *handler.CompleteProfileHandler,
	verifier *supabase.Verifier,
) {
	requireAuth := middleware.RequireAuth(verifier)
	mux.Handle("POST /api/v1/usuarios", requireAuth(handler.Adapt(createUser, usersTimeout)))
	mux.Handle("PATCH /api/v1/usuarios/me", requireAuth(handler.Adapt(completeProfile, usersTimeout)))
}
