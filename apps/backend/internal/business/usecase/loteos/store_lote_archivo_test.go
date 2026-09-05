package loteos_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

const archivoLoteID = "33333333-3333-3333-3333-333333333333"

func loteArchivoInput(loteoID, loteID, categoria, mimeType, content string) loteos.StoreLoteArchivoInput {
	return loteos.StoreLoteArchivoInput{
		LoteoID:   loteoID,
		LoteID:    loteID,
		Categoria: categoria,
		FileName:  "plano.pdf",
		MimeType:  mimeType,
		Content:   bytes.NewReader([]byte(content)),
		Size:      int64(len(content)),
	}
}

func TestStoreLoteArchivoStoresBytesThenRecordsTheFile(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteArchivo(repository, storage)

	const content = "%PDF-1.4 fake"
	archivo, err := useCase.Execute(
		context.Background(), administrador(),
		loteArchivoInput(archivoLoteoID, archivoLoteID, "plano", "application/pdf", content),
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantPrefix := "loteos/" + archivoLoteoID + "/lotes/" + archivoLoteID + "/archivos/"
	if !strings.HasPrefix(archivo.StorageKey, wantPrefix) {
		t.Fatalf("StorageKey = %q, want prefix %q", archivo.StorageKey, wantPrefix)
	}
	if repository.RecordedLoteArchivoLoteoID != archivoLoteoID || repository.RecordedLoteArchivoLoteID != archivoLoteID {
		t.Fatalf("recorded ids = (%q, %q), want (%q, %q)",
			repository.RecordedLoteArchivoLoteoID, repository.RecordedLoteArchivoLoteID, archivoLoteoID, archivoLoteID)
	}
}

func TestStoreLoteArchivoRejectsAnUnassignedAgrimensor(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, Assigned: false}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteArchivo(repository, storage)

	_, err := useCase.Execute(
		context.Background(), agrimensor(),
		loteArchivoInput(archivoLoteoID, archivoLoteID, "plano", "application/pdf", "x"),
	)

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
}

func TestStoreLoteArchivoRejectsAFullEntity(t *testing.T) {
	existing := make([]domain.Archivo, domain.MaxArchivosPerEntity)
	repository := &gatewayfake.LoteoRepository{Exists: true, ListLoteArchivosResult: existing}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteArchivo(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(),
		loteArchivoInput(archivoLoteoID, archivoLoteID, "plano", "application/pdf", "x"),
	)

	if !errors.Is(err, domain.ErrTooManyArchivos) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrTooManyArchivos)
	}
}

func TestStoreLoteArchivoPropagatesALoteNotFoundFromRecording(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, RecordLoteArchivoErr: domain.ErrLoteNotFound}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteArchivo(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(),
		loteArchivoInput(archivoLoteoID, archivoLoteID, "plano", "application/pdf", "x"),
	)

	if !errors.Is(err, domain.ErrLoteNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrLoteNotFound)
	}
	if storage.DeleteCalls != 1 {
		t.Fatalf("DeleteCalls = %d, want 1 (the orphaned upload is cleaned up)", storage.DeleteCalls)
	}
}
