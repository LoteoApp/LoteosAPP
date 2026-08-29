package route

import (
	"net/http"
	"time"

	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

const (
	systemTimeout  = 2 * time.Second
	usersTimeout   = 5 * time.Second
	clientsTimeout = 5 * time.Second
	lotesTimeout   = 10 * time.Second

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
	GetSystemInfo          *handler.GetSystemInfoHandler
	CheckDatabaseReadiness *handler.CheckDatabaseReadinessHandler
	CreateUser             *handler.CreateUserHandler
	CompleteProfile        *handler.CompleteProfileHandler
	CreateClient           *handler.CreateClientHandler
	UpdateClient           *handler.UpdateClientHandler
	DeleteClient           *handler.DeleteClientHandler
	ListClients            *handler.ListClientsHandler
	CreateLoteo            *handler.CreateLoteoHandler
	UpdateLote             *handler.UpdateLoteHandler
}

func RegisterRoutes(mux *http.ServeMux, handlers Handlers, verifier *supabase.Verifier) {
	mux.HandleFunc("GET /healthz", handler.Live)
	mux.HandleFunc("GET /readyz", handler.Adapt(handlers.CheckDatabaseReadiness, systemTimeout))
	mux.HandleFunc("GET /api/v1/system", handler.Adapt(handlers.GetSystemInfo, systemTimeout))

	requireAuth := middleware.RequireAuth(verifier)
	mux.Handle("POST /api/v1/usuarios", requireAuth(handler.Adapt(handlers.CreateUser, usersTimeout)))
	mux.Handle("PATCH /api/v1/usuarios/me", requireAuth(handler.Adapt(handlers.CompleteProfile, usersTimeout)))

	mux.Handle("POST /api/v1/clientes", requireAuth(handler.Adapt(handlers.CreateClient, clientsTimeout)))
	mux.Handle("PATCH /api/v1/clientes/{id}", requireAuth(handler.Adapt(handlers.UpdateClient, clientsTimeout)))
	mux.Handle("DELETE /api/v1/clientes/{id}", requireAuth(handler.Adapt(handlers.DeleteClient, clientsTimeout)))
	mux.Handle("GET /api/v1/clientes", requireAuth(handler.Adapt(handlers.ListClients, clientsTimeout)))

	mux.Handle("POST /api/v1/loteos", requireAuth(handler.Adapt(handlers.CreateLoteo, createLoteoTimeout)))
	mux.Handle("PATCH /api/v1/loteos/{loteoId}/lotes/{loteId}", requireAuth(handler.Adapt(handlers.UpdateLote, lotesTimeout)))
}
