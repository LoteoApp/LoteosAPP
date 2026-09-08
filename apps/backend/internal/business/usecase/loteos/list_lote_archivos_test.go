package loteos_test

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func TestListLoteFilesReturnsTheRepositoryResultForAVisibleLoteo(t *testing.T) {
	t.Parallel()

	want := []domain.File{{ID: "archivo-1", Category: "plano"}}
	repository := &gatewayfake.LoteoRepository{
		GetResult:           domain.Loteo{ID: "loteo-1"},
		ListLoteFilesResult: want,
	}
	useCase := loteos.NewListLoteFiles(repository)

	got, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "lote-1")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "archivo-1" {
		t.Fatalf("Execute() = %#v, want %#v", got, want)
	}
	if repository.ListLoteFilesLoteoID != "loteo-1" || repository.ListLoteFilesLoteID != "lote-1" {
		t.Errorf("ids = (%q, %q), want (loteo-1, lote-1)", repository.ListLoteFilesLoteoID, repository.ListLoteFilesLoteID)
	}
}

func TestListLoteFilesReportsNotFoundForALoteoOutsideTheActorsScope(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{GetErr: domain.ErrLoteoNotFound}
	useCase := loteos.NewListLoteFiles(repository)

	_, err := useCase.Execute(context.Background(), actorWith(domain.RolInmobiliaria), "loteo-1", "lote-1")

	if !errors.Is(err, domain.ErrLoteoNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrLoteoNotFound)
	}
	if repository.ListLoteFilesCalls != 0 {
		t.Error("Execute() should not list archivos for a loteo outside the actor's scope")
	}
}

func TestListLoteFilesDeniesAnyOtherActor(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewListLoteFiles(repository)

	_, err := useCase.Execute(context.Background(), loteos.Actor{AuthProviderID: "actor-1"}, "loteo-1", "lote-1")

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
}
