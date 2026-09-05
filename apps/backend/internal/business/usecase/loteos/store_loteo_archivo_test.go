package loteos_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

const archivoLoteoID = "22222222-2222-2222-2222-222222222222"

func loteoArchivoInput(loteoID, categoria, mimeType, content string) loteos.StoreLoteoArchivoInput {
	return loteos.StoreLoteoArchivoInput{
		LoteoID:   loteoID,
		Categoria: categoria,
		FileName:  "foto.jpg",
		MimeType:  mimeType,
		Content:   bytes.NewReader([]byte(content)),
		Size:      int64(len(content)),
	}
}

func TestStoreLoteoArchivoStoresBytesThenRecordsTheFile(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoArchivo(repository, storage)

	const content = "fake-jpeg-bytes"
	archivo, err := useCase.Execute(
		context.Background(), administrador(), loteoArchivoInput(archivoLoteoID, "foto", "image/jpeg", content),
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantPrefix := "loteos/" + archivoLoteoID + "/archivos/"
	if !strings.HasPrefix(archivo.StorageKey, wantPrefix) {
		t.Fatalf("StorageKey = %q, want prefix %q", archivo.StorageKey, wantPrefix)
	}
	if storage.PutCalls != 1 || repository.RecordLoteoArchivoCalls != 1 {
		t.Fatalf("PutCalls = %d, RecordLoteoArchivoCalls = %d, want 1 and 1", storage.PutCalls, repository.RecordLoteoArchivoCalls)
	}
	if archivo.Categoria != "foto" {
		t.Fatalf("Categoria = %q, want foto", archivo.Categoria)
	}

	stored, ok := storage.Contents(archivo.StorageKey)
	if !ok || string(stored) != content {
		t.Fatalf("stored object = %q (found %v), want %q", stored, ok, content)
	}

	digest := sha256.Sum256([]byte(content))
	if repository.RecordedLoteoArchivo.Sha256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("recorded sha256 = %q, want %q", repository.RecordedLoteoArchivo.Sha256, hex.EncodeToString(digest[:]))
	}
}

func TestStoreLoteoArchivoRejectsRolesOtherThanAdminOrAgrimensor(t *testing.T) {
	for _, rol := range []string{domain.RolAdministrativo, domain.RolEscribano, domain.RolInmobiliaria} {
		t.Run(rol, func(t *testing.T) {
			repository := &gatewayfake.LoteoRepository{Exists: true}
			storage := &gatewayfake.ObjectStorage{}
			useCase := loteos.NewStoreLoteoArchivo(repository, storage)

			actor := loteos.Actor{AuthProviderID: "actor-1", Roles: []string{rol}}
			_, err := useCase.Execute(
				context.Background(), actor, loteoArchivoInput(archivoLoteoID, "foto", "image/jpeg", "x"),
			)

			if !errors.Is(err, domain.ErrNoAutorizado) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
			}
			if storage.PutCalls != 0 {
				t.Fatalf("PutCalls = %d, want 0", storage.PutCalls)
			}
		})
	}
}

func TestStoreLoteoArchivoRejectsAnUnassignedAgrimensor(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, Assigned: false}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoArchivo(repository, storage)

	_, err := useCase.Execute(
		context.Background(), agrimensor(), loteoArchivoInput(archivoLoteoID, "foto", "image/jpeg", "x"),
	)

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
}

func TestStoreLoteoArchivoRejectsAnUnknownCategoria(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoArchivo(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(), loteoArchivoInput(archivoLoteoID, "dxf", "image/jpeg", "x"),
	)

	if !errors.Is(err, domain.ErrInvalidArchivo) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidArchivo)
	}
	if storage.PutCalls != 0 {
		t.Fatalf("PutCalls = %d, want 0", storage.PutCalls)
	}
}

func TestStoreLoteoArchivoRejectsAnUnsupportedMimeType(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoArchivo(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(), loteoArchivoInput(archivoLoteoID, "foto", "application/zip", "x"),
	)

	if !errors.Is(err, domain.ErrInvalidArchivo) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidArchivo)
	}
}

func TestStoreLoteoArchivoRejectsAnOversizedFile(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoArchivo(repository, storage)

	input := loteoArchivoInput(archivoLoteoID, "foto", "image/jpeg", "x")
	input.Size = domain.MaxArchivoFileBytes + 1

	_, err := useCase.Execute(context.Background(), administrador(), input)

	if !errors.Is(err, domain.ErrInvalidArchivo) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidArchivo)
	}
	if storage.PutCalls != 0 {
		t.Fatalf("PutCalls = %d, want 0", storage.PutCalls)
	}
}

func TestStoreLoteoArchivoRejectsAFullEntity(t *testing.T) {
	existing := make([]domain.Archivo, domain.MaxArchivosPerEntity)
	repository := &gatewayfake.LoteoRepository{Exists: true, ListLoteoArchivosResult: existing}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoArchivo(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(), loteoArchivoInput(archivoLoteoID, "foto", "image/jpeg", "x"),
	)

	if !errors.Is(err, domain.ErrTooManyArchivos) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrTooManyArchivos)
	}
	if storage.PutCalls != 0 {
		t.Fatalf("PutCalls = %d, want 0", storage.PutCalls)
	}
}

func TestStoreLoteoArchivoCleansUpTheUploadWhenRecordingFails(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{
		Exists:                true,
		RecordLoteoArchivoErr: errors.New("connection reset"),
	}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoArchivo(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(), loteoArchivoInput(archivoLoteoID, "foto", "image/jpeg", "x"),
	)

	if err == nil {
		t.Fatal("Execute() error = nil, want an error")
	}
	if storage.DeleteCalls != 1 {
		t.Fatalf("DeleteCalls = %d, want 1", storage.DeleteCalls)
	}
}

func TestStoreLoteoArchivoSurfacesARepositoryFailureWhenListingForTheLimit(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, ListLoteoArchivosErr: errors.New("connection reset")}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoArchivo(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(), loteoArchivoInput(archivoLoteoID, "foto", "image/jpeg", "x"),
	)

	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != domain.KindUnavailable {
		t.Fatalf("Execute() error = %v, want an unavailable-kind domain error", err)
	}
	if storage.PutCalls != 0 {
		t.Fatalf("PutCalls = %d, want 0", storage.PutCalls)
	}
}
