package loteos_test

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func TestListLoteArchivosReturnsTheRepositoryResultForAVisibleLoteo(t *testing.T) {
	t.Parallel()

	want := []domain.Archivo{{ID: "archivo-1", Categoria: "plano"}}
	repository := &gatewayfake.LoteoRepository{
		GetResult:              domain.Loteo{ID: "loteo-1"},
		ListLoteArchivosResult: want,
	}
	useCase := loteos.NewListLoteArchivos(repository)

	got, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "lote-1")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "archivo-1" {
		t.Fatalf("Execute() = %#v, want %#v", got, want)
	}
	if repository.ListLoteArchivosLoteoID != "loteo-1" || repository.ListLoteArchivosLoteID != "lote-1" {
		t.Errorf("ids = (%q, %q), want (loteo-1, lote-1)", repository.ListLoteArchivosLoteoID, repository.ListLoteArchivosLoteID)
	}
}

func TestListLoteArchivosReportsNotFoundForALoteoOutsideTheActorsScope(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{GetErr: domain.ErrLoteoNotFound}
	useCase := loteos.NewListLoteArchivos(repository)

	_, err := useCase.Execute(context.Background(), actorWith(domain.RolInmobiliaria), "loteo-1", "lote-1")

	if !errors.Is(err, domain.ErrLoteoNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrLoteoNotFound)
	}
	if repository.ListLoteArchivosCalls != 0 {
		t.Error("Execute() should not list archivos for a loteo outside the actor's scope")
	}
}

func TestListLoteArchivosDeniesAnyOtherActor(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewListLoteArchivos(repository)

	_, err := useCase.Execute(context.Background(), loteos.Actor{AuthProviderID: "actor-1"}, "loteo-1", "lote-1")

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
}
