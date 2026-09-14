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

func loteFileInput(loteoID, loteID, category, mimeType, content string) loteos.StoreLoteFileInput {
	return loteos.StoreLoteFileInput{
		LoteoID:  loteoID,
		LoteID:   loteID,
		Category: category,
		FileName: "plano.pdf",
		MimeType: mimeType,
		Content:  bytes.NewReader([]byte(content)),
		Size:     int64(len(content)),
	}
}

func TestStoreLoteFileStoresBytesThenRecordsTheFile(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteFile(repository, storage)

	const content = "%PDF-1.4 fake"
	file, err := useCase.Execute(
		context.Background(), administrador(),
		loteFileInput(archivoLoteoID, archivoLoteID, "plano", "application/pdf", content),
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantPrefix := "loteos/" + archivoLoteoID + "/lotes/" + archivoLoteID + "/archivos/"
	if !strings.HasPrefix(file.StorageKey, wantPrefix) {
		t.Fatalf("StorageKey = %q, want prefix %q", file.StorageKey, wantPrefix)
	}
	if repository.RecordedLoteFileLoteoID != archivoLoteoID || repository.RecordedLoteFileLoteID != archivoLoteID {
		t.Fatalf("recorded ids = (%q, %q), want (%q, %q)",
			repository.RecordedLoteFileLoteoID, repository.RecordedLoteFileLoteID, archivoLoteoID, archivoLoteID)
	}
}

func TestStoreLoteFileRejectsAnUnassignedAgrimensor(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, Assigned: false}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteFile(repository, storage)

	_, err := useCase.Execute(
		context.Background(), agrimensor(),
		loteFileInput(archivoLoteoID, archivoLoteID, "plano", "application/pdf", "x"),
	)

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
}

func TestStoreLoteFileRejectsAFullEntity(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, RecordLoteFileErr: domain.ErrTooManyFiles}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteFile(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(),
		loteFileInput(archivoLoteoID, archivoLoteID, "plano", "application/pdf", "%PDF-1.4 x"),
	)

	if !errors.Is(err, domain.ErrTooManyFiles) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrTooManyFiles)
	}
	if storage.DeleteCalls != 1 {
		t.Fatalf("DeleteCalls = %d, want 1 (the orphaned upload is cleaned up)", storage.DeleteCalls)
	}
}

func TestStoreLoteFilePropagatesALoteNotFoundFromRecording(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, RecordLoteFileErr: domain.ErrLoteNotFound}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteFile(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(),
		loteFileInput(archivoLoteoID, archivoLoteID, "plano", "application/pdf", "%PDF-1.4 x"),
	)

	if !errors.Is(err, domain.ErrLoteNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrLoteNotFound)
	}
	if storage.DeleteCalls != 1 {
		t.Fatalf("DeleteCalls = %d, want 1 (the orphaned upload is cleaned up)", storage.DeleteCalls)
	}
}

func TestStoreLoteFileRejectsContentThatDoesNotMatchTheDeclaredMimeType(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteFile(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(),
		loteFileInput(archivoLoteoID, archivoLoteID, "plano", "application/pdf", "this is plain text, not a pdf"),
	)

	if !errors.Is(err, domain.ErrInvalidFile) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidFile)
	}
	if storage.PutCalls != 0 {
		t.Fatalf("PutCalls = %d, want 0", storage.PutCalls)
	}
}
