package route

import (
	"net/http"
	"time"

	"loteosapp/backend/internal/business/gateway"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

const (
	usersTimeout   = 5 * time.Second
	clientsTimeout = 5 * time.Second
	lotesTimeout   = 10 * time.Second

	// Registering a loteo writes one row per polygon of a whole cadastral
	// plan in a single transaction, so it gets more room than a request that
	// touches one row.
	createLoteoTimeout = 60 * time.Second

	// Uploading the original DXF streams up to domain.MaxDxfFileBytes to R2
	// before writing one row, so it gets the same room as the alta.
	uploadDxfTimeout = 60 * time.Second

	// MaxHandlerTimeout is the longest any route registered here may take.
	// The HTTP server's own deadlines are derived from it, so a handler that
	// runs to its limit can still write its response instead of having the
	// connection cut after the transaction already committed.
	MaxHandlerTimeout = createLoteoTimeout
)

type Handlers struct {
	CreateUser      *handler.CreateUserHandler
	CompleteProfile *handler.CompleteProfileHandler
	ListUsers       *handler.ListUsersHandler
	UpdateUser      *handler.UpdateUserHandler
	DeactivateUser  *handler.DeactivateUserHandler
	ReactivateUser  *handler.ReactivateUserHandler
	CreateClient    *handler.CreateClientHandler
	UpdateClient    *handler.UpdateClientHandler
	DeleteClient    *handler.DeleteClientHandler
	ListClients     *handler.ListClientsHandler
	CreateLoteo     *handler.CreateLoteoHandler
	StoreLoteoDxf   *handler.StoreLoteoDxfHandler
	UpdateLote      *handler.UpdateLoteHandler
}

// RegisterRoutes wires every route behind both RequireAuth (a valid token)
// and RequireActiveAccount (a usuarios row that isn't given de baja): a baja
// must block a caller right away, not just once their current token expires.
func RegisterRoutes(mux *http.ServeMux, handlers Handlers, verifier *supabase.Verifier, userRepository gateway.UserRepository) {
	requireAuth := middleware.RequireAuth(verifier)
	requireActiveAccount := middleware.RequireActiveAccount(userRepository)
	protected := func(h http.Handler) http.Handler {
		return requireAuth(requireActiveAccount(h))
	}

	mux.Handle("POST /api/v1/usuarios", protected(handler.Adapt(handlers.CreateUser, usersTimeout)))
	mux.Handle("PATCH /api/v1/usuarios/me", protected(handler.Adapt(handlers.CompleteProfile, usersTimeout)))
	mux.Handle("GET /api/v1/usuarios", protected(handler.Adapt(handlers.ListUsers, usersTimeout)))
	mux.Handle("PATCH /api/v1/usuarios/{id}", protected(handler.Adapt(handlers.UpdateUser, usersTimeout)))
	mux.Handle("DELETE /api/v1/usuarios/{id}", protected(handler.Adapt(handlers.DeactivateUser, usersTimeout)))
	mux.Handle("POST /api/v1/usuarios/{id}/reactivar", protected(handler.Adapt(handlers.ReactivateUser, usersTimeout)))

	mux.Handle("POST /api/v1/clientes", protected(handler.Adapt(handlers.CreateClient, clientsTimeout)))
	mux.Handle("PATCH /api/v1/clientes/{id}", protected(handler.Adapt(handlers.UpdateClient, clientsTimeout)))
	mux.Handle("DELETE /api/v1/clientes/{id}", protected(handler.Adapt(handlers.DeleteClient, clientsTimeout)))
	mux.Handle("GET /api/v1/clientes", protected(handler.Adapt(handlers.ListClients, clientsTimeout)))

	mux.Handle("POST /api/v1/loteos", protected(handler.Adapt(handlers.CreateLoteo, createLoteoTimeout)))
	mux.Handle("PUT /api/v1/loteos/{loteoId}/dxf", protected(handler.Adapt(handlers.StoreLoteoDxf, uploadDxfTimeout)))
	mux.Handle("PATCH /api/v1/loteos/{loteoId}/lotes/{loteId}", protected(handler.Adapt(handlers.UpdateLote, lotesTimeout)))
}
