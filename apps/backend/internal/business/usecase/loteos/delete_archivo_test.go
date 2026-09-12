package loteos_test

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func TestDeleteFileDeletesForAnAdministrador(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewDeleteFile(repository)

	if err := useCase.Execute(context.Background(), administrador(), "loteo-1", "archivo-1"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.DeletedFileLoteoID != "loteo-1" || repository.DeletedFileID != "archivo-1" {
		t.Errorf("deleted ids = (%q, %q), want (loteo-1, archivo-1)",
			repository.DeletedFileLoteoID, repository.DeletedFileID)
	}
}

func TestDeleteFileRejectsAnUnassignedAgrimensor(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{Assigned: false}
	useCase := loteos.NewDeleteFile(repository)

	err := useCase.Execute(context.Background(), agrimensor(), "loteo-1", "archivo-1")

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.DeleteFileCalls != 0 {
		t.Error("Execute() should not reach the repository for an unauthorized actor")
	}
}

func TestDeleteFilePropagatesANotFoundFromTheRepository(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{DeleteFileErr: domain.ErrFileNotFound}
	useCase := loteos.NewDeleteFile(repository)

	err := useCase.Execute(context.Background(), administrador(), "loteo-1", "archivo-1")

	if !errors.Is(err, domain.ErrFileNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrFileNotFound)
	}
}
