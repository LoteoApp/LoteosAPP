package dependencies

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/gateway"
	"loteosapp/backend/internal/business/usecase/agencies"
	"loteosapp/backend/internal/business/usecase/clients"
	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/business/usecase/reservations"
	"loteosapp/backend/internal/business/usecase/sales"
	"loteosapp/backend/internal/business/usecase/users"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/email/resend"
	"loteosapp/backend/internal/infrastructure/environments"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
	"loteosapp/backend/internal/infrastructure/storage/r2"
	"loteosapp/backend/internal/infrastructure/worker"
)

type Container struct {
	CreateUserHandler          *handler.CreateUserHandler
	CompleteProfileHandler     *handler.CompleteProfileHandler
	ListUsersHandler           *handler.ListUsersHandler
	UpdateUserHandler          *handler.UpdateUserHandler
	DeactivateUserHandler      *handler.DeactivateUserHandler
	ReactivateUserHandler      *handler.ReactivateUserHandler
	CreateClientHandler        *handler.CreateClientHandler
	UpdateClientHandler        *handler.UpdateClientHandler
	DeleteClientHandler        *handler.DeleteClientHandler
	ListClientsHandler         *handler.ListClientsHandler
	CreateAgencyHandler        *handler.CreateAgencyHandler
	UpdateAgencyHandler        *handler.UpdateAgencyHandler
	DeleteAgencyHandler        *handler.DeleteAgencyHandler
	ListAgenciesHandler        *handler.ListAgenciesHandler
	CreateLoteoHandler         *handler.CreateLoteoHandler
	StoreLoteoDxfHandler       *handler.StoreLoteoDxfHandler
	UpdateLoteHandler          *handler.UpdateLoteHandler
	UpdateManzanaHandler       *handler.UpdateManzanaHandler
	UpdateCalleHandler         *handler.UpdateCalleHandler
	ListLoteosHandler          *handler.ListLoteosHandler
	GetLoteoHandler            *handler.GetLoteoHandler
	StoreLoteoFileHandler      *handler.StoreLoteoFileHandler
	StoreLoteFileHandler       *handler.StoreLoteFileHandler
	ListLoteoFilesHandler      *handler.ListLoteoFilesHandler
	ListLoteFilesHandler       *handler.ListLoteFilesHandler
	GetFileContentHandler      *handler.GetFileContentHandler
	DeleteFileHandler          *handler.DeleteFileHandler
	CreateReservationHandler   *handler.CreateReservationHandler
	ListReservationsHandler    *handler.ListReservationsHandler
	GetReservationHandler      *handler.GetReservationHandler
	ReservationReceiptHandler  *handler.ReservationReceiptHandler
	CancelReservationHandler   *handler.CancelReservationHandler
	CreateSaleHandler          *handler.CreateSaleHandler
	ListSalesHandler           *handler.ListSalesHandler
	GetSaleHandler             *handler.GetSaleHandler
	ListEligibleSellersHandler *handler.ListEligibleSellersHandler
	ResendInviteEmailHandler   *handler.ResendInviteEmailHandler
	TransitionLotState         loteos.TransitionLotState
	ReservationExpiryWorker    *worker.ReservationExpiryWorker
	Pool                       *pgxpool.Pool
	Verifier                   *supabase.Verifier
	ObjectStorage              gateway.ObjectStorage
	UserRepository             gateway.UserRepository
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

	mailer, err := resend.NewClient(resend.Config{
		APIKey:    cfg.Mailer.APIKey,
		FromEmail: cfg.Mailer.FromEmail,
		FromName:  cfg.Mailer.FromName,
	})
	if err != nil {
		pool.Close()
		return nil, err
	}

	inviteRedirectURL := cfg.FrontendOrigin + "/aceptar-invitacion"
	adminClient := supabase.NewAdminClient(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey, inviteRedirectURL)
	userRepo := postgres.NewUserRepository(pool)
	inmobiliariaRepo := postgres.NewAgencyRepository(pool)
	createUserHandler := handler.NewCreateUserHandler(users.NewCreateUser(userRepo, adminClient, inmobiliariaRepo, mailer))
	completeProfileHandler := handler.NewCompleteProfileHandler(users.NewCompleteProfile(userRepo))
	listUsersHandler := handler.NewListUsersHandler(users.NewListUsers(userRepo, adminClient))
	updateUserHandler := handler.NewUpdateUserHandler(users.NewUpdateUser(userRepo))
	deactivateUserHandler := handler.NewDeactivateUserHandler(users.NewDeactivateUser(userRepo))
	reactivateUserHandler := handler.NewReactivateUserHandler(users.NewReactivateUser(userRepo))
	resendInviteEmailHandler := handler.NewResendInviteEmailHandler(users.NewResendInviteEmail(userRepo, adminClient, mailer))

	clienteRepo := postgres.NewClienteRepository(pool)
	createClientHandler := handler.NewCreateClientHandler(clients.NewCreateClient(clienteRepo, userRepo))
	updateClientHandler := handler.NewUpdateClientHandler(clients.NewUpdateClient(clienteRepo, userRepo))
	deleteClientHandler := handler.NewDeleteClientHandler(clients.NewDeleteClient(clienteRepo, userRepo))
	listClientsHandler := handler.NewListClientsHandler(clients.NewListClients(clienteRepo))

	createAgencyHandler := handler.NewCreateAgencyHandler(agencies.NewCreateAgency(inmobiliariaRepo, userRepo))
	updateAgencyHandler := handler.NewUpdateAgencyHandler(agencies.NewUpdateAgency(inmobiliariaRepo, userRepo))
	deleteAgencyHandler := handler.NewDeleteAgencyHandler(agencies.NewDeleteAgency(inmobiliariaRepo, userRepo))
	listAgenciesHandler := handler.NewListAgenciesHandler(agencies.NewListAgencies(inmobiliariaRepo))

	loteoRepo := postgres.NewLoteoRepository(pool)
	lotStateRepo := postgres.NewLotStateRepository(pool)
	createLoteoHandler := handler.NewCreateLoteoHandler(loteos.NewCreateLoteo(loteoRepo))
	storeLoteoDxfHandler := handler.NewStoreLoteoDxfHandler(loteos.NewStoreLoteoDxf(loteoRepo, objectStorage))
	updateLoteHandler := handler.NewUpdateLoteHandler(loteos.NewUpdateLote(loteoRepo))
	updateManzanaHandler := handler.NewUpdateManzanaHandler(loteos.NewUpdateManzana(loteoRepo))
	updateCalleHandler := handler.NewUpdateCalleHandler(loteos.NewUpdateCalle(loteoRepo))
	listLoteosHandler := handler.NewListLoteosHandler(loteos.NewListLoteos(loteoRepo))
	getLoteoHandler := handler.NewGetLoteoHandler(loteos.NewGetLoteo(loteoRepo))
	storeLoteoFileHandler := handler.NewStoreLoteoFileHandler(loteos.NewStoreLoteoFile(loteoRepo, objectStorage))
	storeLoteFileHandler := handler.NewStoreLoteFileHandler(loteos.NewStoreLoteFile(loteoRepo, objectStorage))
	listLoteoFilesHandler := handler.NewListLoteoFilesHandler(loteos.NewListLoteoFiles(loteoRepo))
	listLoteFilesHandler := handler.NewListLoteFilesHandler(loteos.NewListLoteFiles(loteoRepo))
	getFileContentHandler := handler.NewGetFileContentHandler(loteos.NewGetFileContent(loteoRepo, objectStorage))
	deleteFileHandler := handler.NewDeleteFileHandler(loteos.NewDeleteFile(loteoRepo))

	transitionLotState := loteos.NewTransitionLotState(lotStateRepo, userRepo)
	reservationRepo := postgres.NewReservationRepository(pool)
	createReservationHandler := handler.NewCreateReservationHandler(reservations.NewCreateReservation(reservationRepo, userRepo))
	listReservationsHandler := handler.NewListReservationsHandler(reservations.NewListReservations(reservationRepo))
	getReservation := reservations.NewGetReservation(reservationRepo)
	getReservationHandler := handler.NewGetReservationHandler(getReservation)
	reservationReceiptHandler := handler.NewReservationReceiptHandler(reservations.NewGetReservationReceipt(reservationRepo, loteoRepo))
	cancelReservationHandler := handler.NewCancelReservationHandler(reservations.NewCancelReservation(reservationRepo, userRepo))
	listEligibleSellersHandler := handler.NewListEligibleSellersHandler(reservations.NewListEligibleSellers(reservationRepo))

	saleRepo := postgres.NewSaleRepository(pool)
	createSaleHandler := handler.NewCreateSaleHandler(sales.NewCreateSale(saleRepo, userRepo))
	listSalesHandler := handler.NewListSalesHandler(sales.NewListSales(saleRepo))
	getSaleHandler := handler.NewGetSaleHandler(sales.NewGetSale(saleRepo))
	var reservationExpiryWorker *worker.ReservationExpiryWorker
	if cfg.ReservationExpiry.Enabled {
		reservationExpiryWorker = worker.NewReservationExpiryWorker(
			reservations.NewProcessExpirations(reservationRepo),
			cfg.ReservationExpiry.Interval,
			cfg.ReservationExpiry.Batch,
			cfg.ReservationExpiry.Timeout,
		)
	}

	return &Container{
		CreateUserHandler:          createUserHandler,
		CompleteProfileHandler:     completeProfileHandler,
		ListUsersHandler:           listUsersHandler,
		UpdateUserHandler:          updateUserHandler,
		DeactivateUserHandler:      deactivateUserHandler,
		ReactivateUserHandler:      reactivateUserHandler,
		CreateClientHandler:        createClientHandler,
		UpdateClientHandler:        updateClientHandler,
		DeleteClientHandler:        deleteClientHandler,
		ListClientsHandler:         listClientsHandler,
		CreateAgencyHandler:        createAgencyHandler,
		UpdateAgencyHandler:        updateAgencyHandler,
		DeleteAgencyHandler:        deleteAgencyHandler,
		ListAgenciesHandler:        listAgenciesHandler,
		CreateLoteoHandler:         createLoteoHandler,
		StoreLoteoDxfHandler:       storeLoteoDxfHandler,
		UpdateLoteHandler:          updateLoteHandler,
		UpdateManzanaHandler:       updateManzanaHandler,
		UpdateCalleHandler:         updateCalleHandler,
		ListLoteosHandler:          listLoteosHandler,
		GetLoteoHandler:            getLoteoHandler,
		StoreLoteoFileHandler:      storeLoteoFileHandler,
		StoreLoteFileHandler:       storeLoteFileHandler,
		ListLoteoFilesHandler:      listLoteoFilesHandler,
		ListLoteFilesHandler:       listLoteFilesHandler,
		GetFileContentHandler:      getFileContentHandler,
		DeleteFileHandler:          deleteFileHandler,
		CreateReservationHandler:   createReservationHandler,
		ListReservationsHandler:    listReservationsHandler,
		GetReservationHandler:      getReservationHandler,
		ReservationReceiptHandler:  reservationReceiptHandler,
		CancelReservationHandler:   cancelReservationHandler,
		CreateSaleHandler:          createSaleHandler,
		ListSalesHandler:           listSalesHandler,
		GetSaleHandler:             getSaleHandler,
		ListEligibleSellersHandler: listEligibleSellersHandler,
		ResendInviteEmailHandler:   resendInviteEmailHandler,
		TransitionLotState:         transitionLotState,
		ReservationExpiryWorker:    reservationExpiryWorker,
		Pool:                       pool,
		Verifier:                   verifier,
		ObjectStorage:              objectStorage,
		UserRepository:             userRepo,
	}, nil
}
