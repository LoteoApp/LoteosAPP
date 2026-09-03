package loteos_test

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func validCalleData() domain.CalleData {
	return domain.CalleData{Name: "Los Álamos", Type: domain.CalleTypeAsfalto}
}

func TestUpdateCalleLetsAnAdministradorEditAnyLoteo(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewUpdateCalle(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "ca-1", validCalleData())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.UpdateCalleCalls != 1 {
		t.Errorf("UpdateCalleCalls = %d, want 1", repository.UpdateCalleCalls)
	}
	if repository.AssignedCall != 0 {
		t.Error("Execute() should not look up assignments for an administrador")
	}
}

func TestUpdateCalleLetsAnAgrimensorEditAnAssignedLoteo(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Assigned: true}
	useCase := loteos.NewUpdateCalle(repository)

	actor := loteos.Actor{AuthProviderID: "actor-2", Roles: []string{domain.RolAgrimensor}}
	if _, err := useCase.Execute(context.Background(), actor, "loteo-1", "ca-1", validCalleData()); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestUpdateCalleRejectsAnAgrimensorOnAnUnassignedLoteo(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Assigned: false}
	useCase := loteos.NewUpdateCalle(repository)

	actor := loteos.Actor{AuthProviderID: "actor-2", Roles: []string{domain.RolAgrimensor}}
	_, err := useCase.Execute(context.Background(), actor, "loteo-1", "ca-1", validCalleData())
	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.UpdateCalleCalls != 0 {
		t.Error("Execute() should not write a calle of a loteo the agrimensor doesn't have assigned")
	}
}

func TestUpdateCalleAuthorizesBeforeItValidates(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewUpdateCalle(repository)

	actor := loteos.Actor{AuthProviderID: "actor-3", Roles: []string{domain.RolInmobiliaria}}
	_, err := useCase.Execute(context.Background(), actor, "loteo-1", "ca-1", domain.CalleData{})
	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
}

func TestUpdateCalleRejectsInvalidData(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewUpdateCalle(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "ca-1", domain.CalleData{Name: "  "})
	if !errors.Is(err, domain.ErrInvalidCalleName) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidCalleName)
	}
	if repository.UpdateCalleCalls != 0 {
		t.Error("Execute() should not reach the repository with invalid data")
	}
}

func TestUpdateCalleNormalizesTextBeforePersisting(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewUpdateCalle(repository)

	if _, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "ca-1", domain.CalleData{
		Name: "  Los Álamos  ", Type: "  Asfalto  ",
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := repository.ReceivedCalleData
	if got.Name != "Los Álamos" {
		t.Errorf("Name = %q, want trimmed", got.Name)
	}
	if got.Type != domain.CalleTypeAsfalto {
		t.Errorf("Type = %q, want lowercase", got.Type)
	}
}

func TestUpdateCalleReturnsTheRepositoryError(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{UpdateCalleErr: domain.ErrCalleNotFound}
	useCase := loteos.NewUpdateCalle(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "ca-1", validCalleData())
	if !errors.Is(err, domain.ErrCalleNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrCalleNotFound)
	}
}

func TestUpdateCalleReportsAnUnexpectedRepositoryFailureAsUnavailable(t *testing.T) {
	cause := errors.New("connection refused")
	repository := &gatewayfake.LoteoRepository{UpdateCalleErr: cause}
	useCase := loteos.NewUpdateCalle(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "ca-1", validCalleData())
	assertUnavailable(t, err, cause)
}
