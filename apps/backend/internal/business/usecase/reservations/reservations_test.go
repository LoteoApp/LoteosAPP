package reservations_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/reservations"
)

type fixedClock struct{ now time.Time }

func (clock fixedClock) Now() time.Time { return clock.now }

func activeAdmin() domain.Usuario {
	return domain.Usuario{ID: "actor-id", AuthProviderID: "actor-subject", Rol: domain.RolAdministrador}
}

func TestCreateReservationNormalizesAndUsesTheAuthoritativeClock(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.ReservationRepository{}
	created := time.Date(2026, 9, 6, 15, 0, 0, 0, time.FixedZone("ART", -3*60*60))
	useCase := reservations.NewCreateReservation(repository, users, fixedClock{now: created})

	reservation, err := useCase.Execute(context.Background(), reservations.CreateReservationInput{
		Actor:          reservations.Actor{AuthProviderID: "actor-subject", Roles: []string{domain.RolAdministrador}},
		LoteoID:        " loteo-id ",
		LoteID:         " lote-id ",
		ClienteID:      " cliente-id ",
		VendedorID:     " seller-id ",
		IdempotencyKey: " request-1 ",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if reservation.ID != "" {
		t.Errorf("unexpected fake result = %#v", reservation)
	}
	command := repository.CreateCommand
	if command.LoteoID != "loteo-id" || command.LoteID != "lote-id" || command.ClienteID != "cliente-id" || command.VendedorID != "seller-id" {
		t.Errorf("normalized command = %#v", command)
	}
	if !command.CreatedAt.Equal(created.UTC()) {
		t.Errorf("created at = %s, want %s", command.CreatedAt, created.UTC())
	}
	if command.CreatedAt.Add(domain.ReservationDuration).Sub(command.CreatedAt) != 360*time.Hour {
		t.Error("reservation should expire after exactly 360 hours")
	}
	if command.IdempotencyPayloadHash == "" || len(command.IdempotencyPayloadHash) != 64 {
		t.Errorf("payload hash = %q", command.IdempotencyPayloadHash)
	}
}

func TestCreateReservationInmobiliariaCannotChooseAnotherSeller(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{
		ID: "agency-user", AuthProviderID: "agency-subject", Rol: domain.RolInmobiliaria,
	}}
	repository := &gatewayfake.ReservationRepository{}
	useCase := reservations.NewCreateReservation(repository, users, fixedClock{})

	_, err := useCase.Execute(context.Background(), reservations.CreateReservationInput{
		Actor:   reservations.Actor{AuthProviderID: "agency-subject", Roles: []string{domain.RolInmobiliaria}},
		LoteoID: "loteo-id", LoteID: "lote-id", ClienteID: "cliente-id",
		VendedorID: "someone-else", IdempotencyKey: "key",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.CreateCommand.VendedorID != "agency-user" || !repository.CreateCommand.SellerIsActor {
		t.Errorf("seller command = %#v", repository.CreateCommand)
	}
}

func TestCreateReservationRejectsUnauthorizedAndMissingIdempotency(t *testing.T) {
	users := &gatewayfake.UserRepository{}
	repository := &gatewayfake.ReservationRepository{}
	useCase := reservations.NewCreateReservation(repository, users)

	_, err := useCase.Execute(context.Background(), reservations.CreateReservationInput{
		Actor:          reservations.Actor{AuthProviderID: "subject", Roles: []string{domain.RolAgrimensor}},
		IdempotencyKey: "key",
	})
	if !errors.Is(err, domain.ErrNoAutorizado) || users.FindByAuthProviderIDCalls != 0 {
		t.Fatalf("unauthorized error = %v, lookups = %d", err, users.FindByAuthProviderIDCalls)
	}
	_, err = useCase.Execute(context.Background(), reservations.CreateReservationInput{
		Actor:   reservations.Actor{AuthProviderID: "subject", Roles: []string{domain.RolAdministrador}},
		LoteoID: "loteo", LoteID: "lote", ClienteID: "cliente", VendedorID: "seller",
	})
	if !errors.Is(err, domain.ErrReservationIdempotencyRequired) {
		t.Fatalf("missing key error = %v", err)
	}
}

func TestListAndCancelApplyScopeAndReason(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.ReservationRepository{}
	list := reservations.NewListReservations(repository)
	_, err := list.Execute(context.Background(), reservations.ListReservationsInput{
		Actor:  reservations.Actor{AuthProviderID: "actor-subject", Roles: []string{domain.RolAdministrativo}},
		Filter: domain.ReservationListFilter{Page: 2, Limit: 10},
	})
	if err != nil || repository.ListFilter.Page != 2 || repository.ListScope.AssigneeAuthProviderID != nil {
		t.Fatalf("list = %#v, err = %v", repository.ListFilter, err)
	}
	cancel := reservations.NewCancelReservation(repository, users, fixedClock{})
	_, err = cancel.Execute(context.Background(), reservations.CancelReservationInput{
		Actor:         reservations.Actor{AuthProviderID: "actor-subject", Roles: []string{domain.RolAdministrador}},
		ReservationID: " reservation-id ", Reason: " Cliente desistió ",
	})
	if err != nil {
		t.Fatalf("cancel error = %v", err)
	}
	if repository.CancelCommand.ReservationID != "reservation-id" || repository.CancelCommand.Reason != "Cliente desistió" {
		t.Errorf("cancel command = %#v", repository.CancelCommand)
	}
}

func TestProcessExpirationsPassesClockAndLimit(t *testing.T) {
	repository := &gatewayfake.ReservationRepository{}
	now := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	useCase := reservations.NewProcessExpirations(repository, fixedClock{now: now})
	if _, err := useCase.Execute(context.Background(), 8); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.ExpireLimit != 8 || !repository.ExpireNow.Equal(now) {
		t.Errorf("expiration input = %d/%s", repository.ExpireLimit, repository.ExpireNow)
	}
}

func TestReservationUseCasesRejectInvalidActorsAndMapRepositoryErrors(t *testing.T) {
	ctx := context.Background()
	validInput := reservations.CreateReservationInput{
		Actor:   reservations.Actor{AuthProviderID: "subject", Roles: []string{domain.RolAdministrador}},
		LoteoID: "loteo", LoteID: "lote", ClienteID: "cliente", VendedorID: "seller", IdempotencyKey: "key",
	}

	t.Run("create validates input and local actor", func(t *testing.T) {
		repository := &gatewayfake.ReservationRepository{}
		users := &gatewayfake.UserRepository{}
		useCase := reservations.NewCreateReservation(repository, users)

		if _, err := useCase.Execute(ctx, reservations.CreateReservationInput{Actor: validInput.Actor}); !errors.Is(err, domain.ErrReservationIdempotencyRequired) {
			t.Fatalf("missing idempotency error = %v", err)
		}
		missingLote := validInput
		missingLote.IdempotencyKey = "key-2"
		missingLote.LoteID = " "
		if _, err := useCase.Execute(ctx, missingLote); !errors.Is(err, domain.ErrLoteNotFound) {
			t.Fatalf("missing lot error = %v", err)
		}
		missingClient := validInput
		missingClient.IdempotencyKey = "key-3"
		missingClient.ClienteID = " "
		if _, err := useCase.Execute(ctx, missingClient); !errors.Is(err, domain.ErrReservationInvalidClient) {
			t.Fatalf("missing client error = %v", err)
		}
		users.FindByAuthProviderIDResult = activeAdmin()
		missingSeller := validInput
		missingSeller.IdempotencyKey = "key-4"
		missingSeller.VendedorID = " "
		if _, err := useCase.Execute(ctx, missingSeller); !errors.Is(err, domain.ErrReservationSellerRequired) {
			t.Fatalf("missing seller error = %v", err)
		}

		users.FindByAuthProviderIDErr = domain.ErrUsuarioNoEncontrado
		if _, err := useCase.Execute(ctx, validInput); !errors.Is(err, domain.ErrActorNoAprovisionado) {
			t.Fatalf("unprovisioned actor error = %v", err)
		}
		users.FindByAuthProviderIDErr = nil
		users.FindByAuthProviderIDResult = domain.Usuario{ID: "inactive", Rol: domain.RolAdministrador, FechaBaja: timePtr(time.Now())}
		if _, err := useCase.Execute(ctx, validInput); !errors.Is(err, domain.ErrCuentaInactiva) {
			t.Fatalf("inactive actor error = %v", err)
		}
		users.FindByAuthProviderIDResult = domain.Usuario{ID: "agrimensor", Rol: domain.RolAgrimensor}
		if _, err := useCase.Execute(ctx, validInput); !errors.Is(err, domain.ErrNoAutorizado) {
			t.Fatalf("role mismatch error = %v", err)
		}
	})

	t.Run("create maps persistence failures", func(t *testing.T) {
		users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
		cause := errors.New("database unavailable")
		repository := &gatewayfake.ReservationRepository{CreateErr: cause}
		useCase := reservations.NewCreateReservation(repository, users, fixedClock{})
		_, err := useCase.Execute(ctx, validInput)
		if !errors.Is(err, domain.ErrDatabaseUnavailable) || !errors.Is(err, cause) {
			t.Fatalf("mapped create error = %v", err)
		}
	})
}

func TestReservationReadUseCasesApplyScopeAndMapFailures(t *testing.T) {
	ctx := context.Background()
	inmobiliaria := reservations.Actor{AuthProviderID: "agency-subject", Roles: []string{domain.RolInmobiliaria}}

	t.Run("list and get", func(t *testing.T) {
		repository := &gatewayfake.ReservationRepository{
			ListErr: errors.New("list failed"),
			GetErr:  errors.New("get failed"),
		}
		list := reservations.NewListReservations(repository)
		if _, err := list.Execute(ctx, reservations.ListReservationsInput{Actor: inmobiliaria, Filter: domain.ReservationListFilter{States: []domain.ReservationState{"bad"}}}); !errors.Is(err, domain.ErrReservationInvalidState) {
			t.Fatalf("invalid filter error = %v", err)
		}
		if _, err := list.Execute(ctx, reservations.ListReservationsInput{Actor: inmobiliaria}); !errors.Is(err, domain.ErrDatabaseUnavailable) {
			t.Fatalf("mapped list error = %v", err)
		}
		if repository.ListScope.AssigneeAuthProviderID == nil || !repository.ListScope.ByAgencyAssignment {
			t.Fatalf("list scope = %#v", repository.ListScope)
		}

		get := reservations.NewGetReservation(repository)
		if _, err := get.Execute(ctx, inmobiliaria, " "); !errors.Is(err, domain.ErrReservationNotFound) {
			t.Fatalf("empty id error = %v", err)
		}
		if _, err := get.Execute(ctx, inmobiliaria, "reservation-id"); !errors.Is(err, domain.ErrDatabaseUnavailable) {
			t.Fatalf("mapped get error = %v", err)
		}
		if repository.GetID != "reservation-id" || repository.GetScope.AssigneeAuthProviderID == nil {
			t.Fatalf("get request = %#v/%#v", repository.GetID, repository.GetScope)
		}
	})

	t.Run("unsupported actors are denied", func(t *testing.T) {
		unsupported := reservations.Actor{AuthProviderID: "subject", Roles: []string{domain.RolAgrimensor}}
		repository := &gatewayfake.ReservationRepository{}
		if _, err := reservations.NewListReservations(repository).Execute(ctx, reservations.ListReservationsInput{Actor: unsupported}); !errors.Is(err, domain.ErrNoAutorizado) {
			t.Fatalf("list authorization error = %v", err)
		}
		if _, err := reservations.NewGetReservation(repository).Execute(ctx, unsupported, "reservation-id"); !errors.Is(err, domain.ErrNoAutorizado) {
			t.Fatalf("get authorization error = %v", err)
		}
	})
}

func TestCancelAndSellerUseCasesValidateInputsAndMapFailures(t *testing.T) {
	ctx := context.Background()
	actor := reservations.Actor{AuthProviderID: "subject", Roles: []string{domain.RolAdministrador}}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}

	t.Run("cancel", func(t *testing.T) {
		repository := &gatewayfake.ReservationRepository{}
		useCase := reservations.NewCancelReservation(repository, users, fixedClock{})
		if _, err := useCase.Execute(ctx, reservations.CancelReservationInput{Actor: actor, ReservationID: "id"}); !errors.Is(err, domain.ErrReservationReasonRequired) {
			t.Fatalf("missing reason error = %v", err)
		}
		if _, err := useCase.Execute(ctx, reservations.CancelReservationInput{Actor: actor, ReservationID: "id", Reason: string(make([]rune, domain.MaxReservationReasonSize+1))}); !errors.Is(err, domain.ErrReservationReasonTooLong) {
			t.Fatalf("long reason error = %v", err)
		}
		if _, err := useCase.Execute(ctx, reservations.CancelReservationInput{Actor: actor, Reason: "valid"}); !errors.Is(err, domain.ErrReservationNotFound) {
			t.Fatalf("missing reservation id error = %v", err)
		}
		cause := errors.New("cancel failed")
		repository.CancelErr = cause
		if _, err := useCase.Execute(ctx, reservations.CancelReservationInput{Actor: actor, ReservationID: "id", Reason: "valid"}); !errors.Is(err, domain.ErrDatabaseUnavailable) || !errors.Is(err, cause) {
			t.Fatalf("mapped cancel error = %v", err)
		}
	})

	t.Run("sellers", func(t *testing.T) {
		repository := &gatewayfake.ReservationRepository{}
		useCase := reservations.NewListEligibleSellers(repository)
		if _, err := useCase.Execute(ctx, reservations.ListEligibleSellersInput{Actor: actor}); !errors.Is(err, domain.ErrLoteoNotFound) {
			t.Fatalf("missing loteo error = %v", err)
		}
		cause := errors.New("catalog failed")
		repository.SellersErr = cause
		if _, err := useCase.Execute(ctx, reservations.ListEligibleSellersInput{Actor: actor, LoteoID: " loteo-id "}); !errors.Is(err, domain.ErrDatabaseUnavailable) || !errors.Is(err, cause) {
			t.Fatalf("mapped sellers error = %v", err)
		}
		if repository.SellersLoteo != "loteo-id" || repository.SellersScope.AssigneeAuthProviderID != nil {
			t.Fatalf("seller request = %#v/%#v", repository.SellersLoteo, repository.SellersScope)
		}
	})
}

func TestProcessExpirationsUsesDefaultLimitAndMapsFailures(t *testing.T) {
	repository := &gatewayfake.ReservationRepository{ExpireErr: errors.New("expiration failed")}
	useCase := reservations.NewProcessExpirations(repository, fixedClock{})
	_, err := useCase.Execute(context.Background(), 0)
	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("mapped expiration error = %v", err)
	}
	if repository.ExpireLimit != domain.DefaultReservationLimit {
		t.Fatalf("default limit = %d", repository.ExpireLimit)
	}
}

func timePtr(value time.Time) *time.Time { return &value }
