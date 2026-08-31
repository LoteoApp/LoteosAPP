package route

import (
	"net/http"
	"time"

	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

const (
	usersTimeout     = 5 * time.Second
	surveyorsTimeout = 5 * time.Second
	lotesTimeout     = 10 * time.Second

	// Registering a loteo writes one row per polygon of a whole cadastral
	// plan in a single transaction, so it gets more room than a request that
	// touches one row.
	createLoteoTimeout = 60 * time.Second

	// MaxHandlerTimeout is the longest any route registered here may take.
	// The HTTP server's own deadlines are derived from it, so a handler that
	// runs to its limit can still write its response instead of having the
	// connection cut after the transaction already committed.
	MaxHandlerTimeout = createLoteoTimeout
)

type Handlers struct {
	CreateUser         *handler.CreateUserHandler
	CompleteProfile    *handler.CompleteProfileHandler
	CreateSurveyor     *handler.CreateSurveyorHandler
	ListSurveyors      *handler.ListSurveyorsHandler
	UpdateSurveyor     *handler.UpdateSurveyorHandler
	DeactivateSurveyor *handler.DeactivateSurveyorHandler
	CreateLoteo        *handler.CreateLoteoHandler
	UpdateLote         *handler.UpdateLoteHandler
}

func RegisterRoutes(mux *http.ServeMux, handlers Handlers, verifier *supabase.Verifier) {
	requireAuth := middleware.RequireAuth(verifier)
	mux.Handle("POST /api/v1/usuarios", requireAuth(handler.Adapt(handlers.CreateUser, usersTimeout)))
	mux.Handle("PATCH /api/v1/usuarios/me", requireAuth(handler.Adapt(handlers.CompleteProfile, usersTimeout)))

	mux.Handle("POST /api/v1/agrimensores", requireAuth(handler.Adapt(handlers.CreateSurveyor, surveyorsTimeout)))
	mux.Handle("GET /api/v1/agrimensores", requireAuth(handler.Adapt(handlers.ListSurveyors, surveyorsTimeout)))
	mux.Handle("PATCH /api/v1/agrimensores/{id}", requireAuth(handler.Adapt(handlers.UpdateSurveyor, surveyorsTimeout)))
	mux.Handle("DELETE /api/v1/agrimensores/{id}", requireAuth(handler.Adapt(handlers.DeactivateSurveyor, surveyorsTimeout)))

	mux.Handle("POST /api/v1/loteos", requireAuth(handler.Adapt(handlers.CreateLoteo, createLoteoTimeout)))
	mux.Handle("PATCH /api/v1/loteos/{loteoId}/lotes/{loteId}", requireAuth(handler.Adapt(handlers.UpdateLote, lotesTimeout)))
}
