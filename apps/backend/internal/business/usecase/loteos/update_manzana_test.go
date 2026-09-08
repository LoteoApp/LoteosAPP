package loteos_test

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func validManzanaData() domain.ManzanaData {
	return domain.ManzanaData{Number: "A", HasWater: true, CalleIDs: []string{"calle-1"}}
}

func TestUpdateManzanaLetsAnAdministradorEditAnyLoteo(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewUpdateManzana(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "mz-1", validManzanaData())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.UpdateManzanaCalls != 1 {
		t.Errorf("UpdateManzanaCalls = %d, want 1", repository.UpdateManzanaCalls)
	}
	if repository.AssignedCall != 0 {
		t.Error("Execute() should not look up assignments for an administrador")
	}
}

func TestUpdateManzanaLetsAnAgrimensorEditAnAssignedLoteo(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Assigned: true}
	useCase := loteos.NewUpdateManzana(repository)

	actor := loteos.Actor{AuthProviderID: "actor-2", Roles: []string{domain.RolAgrimensor}}
	if _, err := useCase.Execute(context.Background(), actor, "loteo-1", "mz-1", validManzanaData()); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestUpdateManzanaRejectsAnAgrimensorOnAnUnassignedLoteo(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Assigned: false}
	useCase := loteos.NewUpdateManzana(repository)

	actor := loteos.Actor{AuthProviderID: "actor-2", Roles: []string{domain.RolAgrimensor}}
	_, err := useCase.Execute(context.Background(), actor, "loteo-1", "mz-1", validManzanaData())
	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.UpdateManzanaCalls != 0 {
		t.Error("Execute() should not write a manzana of a loteo the agrimensor doesn't have assigned")
	}
}

func TestUpdateManzanaAuthorizesBeforeItValidates(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewUpdateManzana(repository)

	actor := loteos.Actor{AuthProviderID: "actor-3", Roles: []string{domain.RolInmobiliaria}}
	_, err := useCase.Execute(context.Background(), actor, "loteo-1", "mz-1", domain.ManzanaData{})
	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
}

func TestUpdateManzanaRejectsInvalidData(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewUpdateManzana(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "mz-1", domain.ManzanaData{Number: "  "})
	if !errors.Is(err, domain.ErrInvalidManzanaNumber) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidManzanaNumber)
	}
	if repository.UpdateManzanaCalls != 0 {
		t.Error("Execute() should not reach the repository with invalid data")
	}
}

func TestUpdateManzanaNormalizesTextBeforePersisting(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewUpdateManzana(repository)

	if _, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "mz-1", domain.ManzanaData{
		Number: "  A  ", CalleIDs: []string{"  calle-1  ", "", "calle-2"},
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := repository.ReceivedManzanaData
	if got.Number != "A" {
		t.Errorf("Number = %q, want trimmed", got.Number)
	}
	if len(got.CalleIDs) != 2 || got.CalleIDs[0] != "calle-1" || got.CalleIDs[1] != "calle-2" {
		t.Errorf("CalleIDs = %#v", got.CalleIDs)
	}
}

func TestUpdateManzanaReturnsTheRepositoryError(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{UpdateManzanaErr: domain.ErrManzanaNotFound}
	useCase := loteos.NewUpdateManzana(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "mz-1", validManzanaData())
	if !errors.Is(err, domain.ErrManzanaNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrManzanaNotFound)
	}
}

func TestUpdateManzanaReportsAnUnexpectedRepositoryFailureAsUnavailable(t *testing.T) {
	cause := errors.New("connection refused")
	repository := &gatewayfake.LoteoRepository{UpdateManzanaErr: cause}
	useCase := loteos.NewUpdateManzana(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "mz-1", validManzanaData())
	assertUnavailable(t, err, cause)
}
