package dependencies

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/gateway"
	"loteosapp/backend/internal/business/usecase/agencies"
	"loteosapp/backend/internal/business/usecase/clients"
	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/business/usecase/users"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/environments"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
	"loteosapp/backend/internal/infrastructure/storage/r2"
)

type Container struct {
	CreateUserHandler      *handler.CreateUserHandler
	CompleteProfileHandler *handler.CompleteProfileHandler
	CreateClientHandler    *handler.CreateClientHandler
	UpdateClientHandler    *handler.UpdateClientHandler
	DeleteClientHandler    *handler.DeleteClientHandler
	ListClientsHandler     *handler.ListClientsHandler
	CreateAgencyHandler    *handler.CreateAgencyHandler
	UpdateAgencyHandler    *handler.UpdateAgencyHandler
	DeleteAgencyHandler    *handler.DeleteAgencyHandler
	ListAgenciesHandler    *handler.ListAgenciesHandler
	CreateLoteoHandler     *handler.CreateLoteoHandler
	StoreLoteoDxfHandler   *handler.StoreLoteoDxfHandler
	UpdateLoteHandler      *handler.UpdateLoteHandler
	ListLoteosHandler      *handler.ListLoteosHandler
	GetLoteoHandler        *handler.GetLoteoHandler
	Pool                   *pgxpool.Pool
	Verifier               *supabase.Verifier
	ObjectStorage          gateway.ObjectStorage
}

func New(ctx context.Context, cfg environments.Server) (*Container, error) {
	pool, err := postgres.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	verifier, err := supabase.NewVerifier(ctx, cfg.SupabaseURL)
	if err != nil {
		pool.Close()
		return nil, err
	}

	objectStorage, err := r2.NewClient(r2.Config{
		Endpoint:        cfg.Storage.Endpoint,
		Bucket:          cfg.Storage.Bucket,
		AccessKeyID:     cfg.Storage.AccessKeyID,
		SecretAccessKey: cfg.Storage.SecretAccessKey,
	})
	if err != nil {
		pool.Close()
		return nil, err
	}

	adminClient := supabase.NewAdminClient(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)
	userRepo := postgres.NewUserRepository(pool)
	createUserHandler := handler.NewCreateUserHandler(users.NewCreateUser(userRepo, adminClient))
	completeProfileHandler := handler.NewCompleteProfileHandler(users.NewCompleteProfile(userRepo))

	clienteRepo := postgres.NewClienteRepository(pool)
	createClientHandler := handler.NewCreateClientHandler(clients.NewCreateClient(clienteRepo, userRepo))
	updateClientHandler := handler.NewUpdateClientHandler(clients.NewUpdateClient(clienteRepo, userRepo))
	deleteClientHandler := handler.NewDeleteClientHandler(clients.NewDeleteClient(clienteRepo, userRepo))
	listClientsHandler := handler.NewListClientsHandler(clients.NewListClients(clienteRepo))

	inmobiliariaRepo := postgres.NewAgencyRepository(pool)
	createAgencyHandler := handler.NewCreateAgencyHandler(agencies.NewCreateAgency(inmobiliariaRepo, userRepo))
	updateAgencyHandler := handler.NewUpdateAgencyHandler(agencies.NewUpdateAgency(inmobiliariaRepo, userRepo))
	deleteAgencyHandler := handler.NewDeleteAgencyHandler(agencies.NewDeleteAgency(inmobiliariaRepo, userRepo))
	listAgenciesHandler := handler.NewListAgenciesHandler(agencies.NewListAgencies(inmobiliariaRepo))

	loteoRepo := postgres.NewLoteoRepository(pool)
	createLoteoHandler := handler.NewCreateLoteoHandler(loteos.NewCreateLoteo(loteoRepo))
	storeLoteoDxfHandler := handler.NewStoreLoteoDxfHandler(loteos.NewStoreLoteoDxf(loteoRepo, objectStorage))
	updateLoteHandler := handler.NewUpdateLoteHandler(loteos.NewUpdateLote(loteoRepo))
	listLoteosHandler := handler.NewListLoteosHandler(loteos.NewListLoteos(loteoRepo))
	getLoteoHandler := handler.NewGetLoteoHandler(loteos.NewGetLoteo(loteoRepo))

	return &Container{
		CreateUserHandler:      createUserHandler,
		CompleteProfileHandler: completeProfileHandler,
		CreateClientHandler:    createClientHandler,
		UpdateClientHandler:    updateClientHandler,
		DeleteClientHandler:    deleteClientHandler,
		ListClientsHandler:     listClientsHandler,
		CreateAgencyHandler:    createAgencyHandler,
		UpdateAgencyHandler:    updateAgencyHandler,
		DeleteAgencyHandler:    deleteAgencyHandler,
		ListAgenciesHandler:    listAgenciesHandler,
		CreateLoteoHandler:     createLoteoHandler,
		StoreLoteoDxfHandler:   storeLoteoDxfHandler,
		UpdateLoteHandler:      updateLoteHandler,
		ListLoteosHandler:      listLoteosHandler,
		GetLoteoHandler:        getLoteoHandler,
		Pool:                   pool,
		Verifier:               verifier,
		ObjectStorage:          objectStorage,
	}, nil
}
