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
)

func RegisterRoutes(
	mux *http.ServeMux,
	getSystemInfo *handler.GetSystemInfoHandler,
	checkDatabaseReadiness *handler.CheckDatabaseReadinessHandler,
	createUser *handler.CreateUserHandler,
	completeProfile *handler.CompleteProfileHandler,
	createClient *handler.CreateClientHandler,
	updateClient *handler.UpdateClientHandler,
	deleteClient *handler.DeleteClientHandler,
	listClients *handler.ListClientsHandler,
	verifier *supabase.Verifier,
) {
	mux.HandleFunc("GET /healthz", handler.Live)
	mux.HandleFunc("GET /readyz", handler.Adapt(checkDatabaseReadiness, systemTimeout))
	mux.HandleFunc("GET /api/v1/system", handler.Adapt(getSystemInfo, systemTimeout))

	requireAuth := middleware.RequireAuth(verifier)
	mux.Handle("POST /api/v1/usuarios", requireAuth(handler.Adapt(createUser, usersTimeout)))
	mux.Handle("PATCH /api/v1/usuarios/me", requireAuth(handler.Adapt(completeProfile, usersTimeout)))

	mux.Handle("POST /api/v1/clientes", requireAuth(handler.Adapt(createClient, clientsTimeout)))
	mux.Handle("PATCH /api/v1/clientes/{id}", requireAuth(handler.Adapt(updateClient, clientsTimeout)))
	mux.Handle("DELETE /api/v1/clientes/{id}", requireAuth(handler.Adapt(deleteClient, clientsTimeout)))
	mux.Handle("GET /api/v1/clientes", requireAuth(handler.Adapt(listClients, clientsTimeout)))
}
