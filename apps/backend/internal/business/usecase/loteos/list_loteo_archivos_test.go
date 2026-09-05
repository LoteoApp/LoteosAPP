package loteos_test

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func TestListLoteoArchivosReturnsTheRepositoryResultForAVisibleLoteo(t *testing.T) {
	t.Parallel()

	want := []domain.Archivo{{ID: "archivo-1", Categoria: "foto"}}
	repository := &gatewayfake.LoteoRepository{
		GetResult:               domain.Loteo{ID: "loteo-1"},
		ListLoteoArchivosResult: want,
	}
	useCase := loteos.NewListLoteoArchivos(repository)

	got, err := useCase.Execute(context.Background(), administrador(), "loteo-1")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "archivo-1" {
		t.Fatalf("Execute() = %#v, want %#v", got, want)
	}
	if repository.ListLoteoArchivosLoteoID != "loteo-1" {
		t.Errorf("loteo id = %q, want loteo-1", repository.ListLoteoArchivosLoteoID)
	}
}

func TestListLoteoArchivosReportsNotFoundForALoteoOutsideTheActorsScope(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{GetErr: domain.ErrLoteoNotFound}
	useCase := loteos.NewListLoteoArchivos(repository)

	_, err := useCase.Execute(context.Background(), actorWith(domain.RolAgrimensor), "loteo-1")

	if !errors.Is(err, domain.ErrLoteoNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrLoteoNotFound)
	}
	if repository.ListLoteoArchivosCalls != 0 {
		t.Error("Execute() should not list archivos for a loteo outside the actor's scope")
	}
}

func TestListLoteoArchivosDeniesAnyOtherActor(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewListLoteoArchivos(repository)

	_, err := useCase.Execute(context.Background(), loteos.Actor{AuthProviderID: "actor-1"}, "loteo-1")

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.GetCalls != 0 {
		t.Error("Execute() should not reach the repository for an unauthorized actor")
	}
}
