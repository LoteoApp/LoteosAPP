package loteos_test

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func TestDeleteArchivoDeletesForAnAdministrador(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewDeleteArchivo(repository)

	if err := useCase.Execute(context.Background(), administrador(), "loteo-1", "archivo-1"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.DeletedArchivoLoteoID != "loteo-1" || repository.DeletedArchivoID != "archivo-1" {
		t.Errorf("deleted ids = (%q, %q), want (loteo-1, archivo-1)",
			repository.DeletedArchivoLoteoID, repository.DeletedArchivoID)
	}
}

func TestDeleteArchivoRejectsAnUnassignedAgrimensor(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{Assigned: false}
	useCase := loteos.NewDeleteArchivo(repository)

	err := useCase.Execute(context.Background(), agrimensor(), "loteo-1", "archivo-1")

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.DeleteArchivoCalls != 0 {
		t.Error("Execute() should not reach the repository for an unauthorized actor")
	}
}

func TestDeleteArchivoPropagatesANotFoundFromTheRepository(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{DeleteArchivoErr: domain.ErrArchivoNotFound}
	useCase := loteos.NewDeleteArchivo(repository)

	err := useCase.Execute(context.Background(), administrador(), "loteo-1", "archivo-1")

	if !errors.Is(err, domain.ErrArchivoNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrArchivoNotFound)
	}
}
