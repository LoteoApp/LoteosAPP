package route

import (
	"net/http"
	"time"

	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

const (
	usersTimeout   = 5 * time.Second
	clientsTimeout = 5 * time.Second
	lotesTimeout   = 10 * time.Second

	// Reading a loteo runs several queries (loteo, manzanas, lotes, calles),
	// so it gets more room than a request that touches one row.
	loteosReadTimeout = 15 * time.Second

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
	CreateClient    *handler.CreateClientHandler
	UpdateClient    *handler.UpdateClientHandler
	DeleteClient    *handler.DeleteClientHandler
	ListClients     *handler.ListClientsHandler
	CreateLoteo     *handler.CreateLoteoHandler
	StoreLoteoDxf   *handler.StoreLoteoDxfHandler
	UpdateLote      *handler.UpdateLoteHandler
	UpdateManzana   *handler.UpdateManzanaHandler
	UpdateCalle     *handler.UpdateCalleHandler
	ListLoteos      *handler.ListLoteosHandler
	GetLoteo        *handler.GetLoteoHandler
}

func RegisterRoutes(mux *http.ServeMux, handlers Handlers, verifier *supabase.Verifier) {
	requireAuth := middleware.RequireAuth(verifier)
	mux.Handle("POST /api/v1/usuarios", requireAuth(handler.Adapt(handlers.CreateUser, usersTimeout)))
	mux.Handle("PATCH /api/v1/usuarios/me", requireAuth(handler.Adapt(handlers.CompleteProfile, usersTimeout)))

	mux.Handle("POST /api/v1/clientes", requireAuth(handler.Adapt(handlers.CreateClient, clientsTimeout)))
	mux.Handle("PATCH /api/v1/clientes/{id}", requireAuth(handler.Adapt(handlers.UpdateClient, clientsTimeout)))
	mux.Handle("DELETE /api/v1/clientes/{id}", requireAuth(handler.Adapt(handlers.DeleteClient, clientsTimeout)))
	mux.Handle("GET /api/v1/clientes", requireAuth(handler.Adapt(handlers.ListClients, clientsTimeout)))

	mux.Handle("POST /api/v1/loteos", requireAuth(handler.Adapt(handlers.CreateLoteo, createLoteoTimeout)))
	mux.Handle("GET /api/v1/loteos", requireAuth(handler.Adapt(handlers.ListLoteos, loteosReadTimeout)))
	mux.Handle("GET /api/v1/loteos/{loteoId}", requireAuth(handler.Adapt(handlers.GetLoteo, loteosReadTimeout)))
	mux.Handle("PUT /api/v1/loteos/{loteoId}/dxf", requireAuth(handler.Adapt(handlers.StoreLoteoDxf, uploadDxfTimeout)))
	mux.Handle("PATCH /api/v1/loteos/{loteoId}/lotes/{loteId}", requireAuth(handler.Adapt(handlers.UpdateLote, lotesTimeout)))
	mux.Handle("PATCH /api/v1/loteos/{loteoId}/manzanas/{manzanaId}", requireAuth(handler.Adapt(handlers.UpdateManzana, lotesTimeout)))
	mux.Handle("PATCH /api/v1/loteos/{loteoId}/calles/{calleId}", requireAuth(handler.Adapt(handlers.UpdateCalle, lotesTimeout)))
}
