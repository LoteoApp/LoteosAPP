package loteos_test

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func validLoteData() domain.LoteData {
	price := 150000.0
	area := 300.5

	return domain.LoteData{
		Number: "12", Price: &price, Currency: "ARS",
		Area: &area, Features: "esquina",
	}
}

func TestUpdateLoteLetsAnAdministradorEditAnyLoteo(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewUpdateLote(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "lote-1", validLoteData())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if repository.UpdateLoteCalls != 1 {
		t.Errorf("UpdateLoteCalls = %d, want 1", repository.UpdateLoteCalls)
	}
	if repository.AssignedCall != 0 {
		t.Error("Execute() should not look up assignments for an administrador")
	}
	if repository.ReceivedLoteoID != "loteo-1" || repository.ReceivedLoteID != "lote-1" {
		t.Errorf("Execute() passed loteo %q and lote %q", repository.ReceivedLoteoID, repository.ReceivedLoteID)
	}
}

func TestUpdateLoteLetsAnAgrimensorEditAnAssignedLoteo(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Assigned: true}
	useCase := loteos.NewUpdateLote(repository)

	actor := loteos.Actor{AuthProviderID: "actor-2", Roles: []string{domain.RolAgrimensor}}
	_, err := useCase.Execute(context.Background(), actor, "loteo-1", "lote-1", validLoteData())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if repository.UpdateLoteCalls != 1 {
		t.Errorf("UpdateLoteCalls = %d, want 1", repository.UpdateLoteCalls)
	}
}

func TestUpdateLoteRejectsAnAgrimensorOnAnUnassignedLoteo(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Assigned: false}
	useCase := loteos.NewUpdateLote(repository)

	actor := loteos.Actor{AuthProviderID: "actor-2", Roles: []string{domain.RolAgrimensor}}
	_, err := useCase.Execute(context.Background(), actor, "loteo-1", "lote-1", validLoteData())

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.UpdateLoteCalls != 0 {
		t.Error("Execute() should not write a lote of a loteo the agrimensor doesn't have assigned")
	}
}

func TestUpdateLoteRejectsEveryOtherRoleWithoutTouchingTheRepository(t *testing.T) {
	for _, rol := range []string{domain.RolAdministrativo, domain.RolEscribano, domain.RolInmobiliaria} {
		t.Run(rol, func(t *testing.T) {
			repository := &gatewayfake.LoteoRepository{Assigned: true}
			useCase := loteos.NewUpdateLote(repository)

			actor := loteos.Actor{AuthProviderID: "actor-3", Roles: []string{rol}}
			_, err := useCase.Execute(context.Background(), actor, "loteo-1", "lote-1", validLoteData())

			if !errors.Is(err, domain.ErrNoAutorizado) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
			}
			if repository.AssignedCall != 0 || repository.UpdateLoteCalls != 0 {
				t.Error("Execute() should reject a role that can never edit lotes without querying anything")
			}
		})
	}
}

func TestUpdateLoteAuthorizesBeforeItValidates(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewUpdateLote(repository)

	actor := loteos.Actor{AuthProviderID: "actor-3", Roles: []string{domain.RolInmobiliaria}}
	_, err := useCase.Execute(context.Background(), actor, "loteo-1", "lote-1", domain.LoteData{Number: ""})

	// An unauthorized caller must not learn which values the endpoint would
	// have rejected.
	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
}

func TestUpdateLoteReturnsTheAssignmentLookupError(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{AssignedErr: errors.New("connection refused")}
	useCase := loteos.NewUpdateLote(repository)

	actor := loteos.Actor{AuthProviderID: "actor-2", Roles: []string{domain.RolAgrimensor}}
	_, err := useCase.Execute(context.Background(), actor, "loteo-1", "lote-1", validLoteData())

	// A failed lookup must not read as "not assigned": that would turn an
	// outage into a permission decision.
	assertUnavailable(t, err, repository.AssignedErr)
	if repository.UpdateLoteCalls != 0 {
		t.Error("Execute() should not write when the assignment lookup failed")
	}
}

func TestUpdateLoteReportsAnUnexpectedRepositoryFailureAsUnavailable(t *testing.T) {
	cause := errors.New("connection refused")
	repository := &gatewayfake.LoteoRepository{UpdateLoteErr: cause}
	useCase := loteos.NewUpdateLote(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "lote-1", validLoteData())

	assertUnavailable(t, err, cause)
}

func TestUpdateLoteRejectsInvalidData(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewUpdateLote(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "lote-1", domain.LoteData{Number: "  "})

	if !errors.Is(err, domain.ErrInvalidLoteNumber) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidLoteNumber)
	}
	if repository.UpdateLoteCalls != 0 {
		t.Error("Execute() should not reach the repository with invalid data")
	}
}

func TestUpdateLoteNormalizesTextBeforePersisting(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewUpdateLote(repository)

	price := 100.0
	data := domain.LoteData{Number: "  12  ", Price: &price, Currency: " ars ", Features: "  esquina  "}
	if _, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "lote-1", data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	received := repository.ReceivedLoteData
	if received.Number != "12" || received.Features != "esquina" {
		t.Errorf("Execute() should trim text fields, got %#v", received)
	}
	if received.Currency != "ARS" {
		t.Errorf("Currency = %q, want the uppercased code", received.Currency)
	}
}

func TestUpdateLoteReturnsTheRepositoryError(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{UpdateLoteErr: domain.ErrLoteNotFound}
	useCase := loteos.NewUpdateLote(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "lote-1", validLoteData())

	if !errors.Is(err, domain.ErrLoteNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrLoteNotFound)
	}
}
