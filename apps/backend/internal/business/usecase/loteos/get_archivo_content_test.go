package loteos_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func TestGetArchivoContentStreamsTheStoredBytes(t *testing.T) {
	t.Parallel()

	storage := &gatewayfake.ObjectStorage{}
	if err := storage.Put(context.Background(), "loteos/loteo-1/archivos/a", bytes.NewReader([]byte("hola")), 4, "image/jpeg"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	repository := &gatewayfake.LoteoRepository{
		GetResult:        domain.Loteo{ID: "loteo-1"},
		GetArchivoResult: domain.Archivo{ID: "archivo-1", StorageKey: "loteos/loteo-1/archivos/a", MimeType: "image/jpeg"},
	}
	useCase := loteos.NewGetArchivoContent(repository, storage)

	content, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "archivo-1")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	defer content.Body.Close()

	body, err := io.ReadAll(content.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(body) != "hola" {
		t.Errorf("body = %q, want %q", body, "hola")
	}
	if content.Archivo.ID != "archivo-1" {
		t.Errorf("Archivo.ID = %q, want archivo-1", content.Archivo.ID)
	}
}

func TestGetArchivoContentReportsNotFoundForALoteoOutsideTheActorsScope(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{GetErr: domain.ErrLoteoNotFound}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewGetArchivoContent(repository, storage)

	_, err := useCase.Execute(context.Background(), actorWith(domain.RolEscribano), "loteo-1", "archivo-1")

	if !errors.Is(err, domain.ErrLoteoNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrLoteoNotFound)
	}
	if storage.GetCalls != 0 {
		t.Error("Execute() should not read storage for a loteo outside the actor's scope")
	}
}

func TestGetArchivoContentPropagatesAnArchivoNotFound(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{
		GetResult:     domain.Loteo{ID: "loteo-1"},
		GetArchivoErr: domain.ErrArchivoNotFound,
	}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewGetArchivoContent(repository, storage)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1", "archivo-1")

	if !errors.Is(err, domain.ErrArchivoNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrArchivoNotFound)
	}
}

func TestGetArchivoContentDeniesAnyOtherActor(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewGetArchivoContent(repository, storage)

	_, err := useCase.Execute(context.Background(), loteos.Actor{AuthProviderID: "actor-1"}, "loteo-1", "archivo-1")

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
}
