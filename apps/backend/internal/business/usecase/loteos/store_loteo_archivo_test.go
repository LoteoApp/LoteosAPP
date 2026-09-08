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

func loteoFileInput(loteoID, category, mimeType, content string) loteos.StoreLoteoFileInput {
	return loteos.StoreLoteoFileInput{
		LoteoID:  loteoID,
		Category: category,
		FileName: "foto.jpg",
		MimeType: mimeType,
		Content:  bytes.NewReader([]byte(content)),
		Size:     int64(len(content)),
	}
}

func TestStoreLoteoFileStoresBytesThenRecordsTheFile(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoFile(repository, storage)

	const content = "\xFF\xD8\xFFfake-jpeg-bytes"
	file, err := useCase.Execute(
		context.Background(), administrador(), loteoFileInput(archivoLoteoID, "foto", "image/jpeg", content),
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantPrefix := "loteos/" + archivoLoteoID + "/archivos/"
	if !strings.HasPrefix(file.StorageKey, wantPrefix) {
		t.Fatalf("StorageKey = %q, want prefix %q", file.StorageKey, wantPrefix)
	}
	if storage.PutCalls != 1 || repository.RecordLoteoFileCalls != 1 {
		t.Fatalf("PutCalls = %d, RecordLoteoFileCalls = %d, want 1 and 1", storage.PutCalls, repository.RecordLoteoFileCalls)
	}
	if file.Category != "foto" {
		t.Fatalf("Category = %q, want foto", file.Category)
	}

	stored, ok := storage.Contents(file.StorageKey)
	if !ok || string(stored) != content {
		t.Fatalf("stored object = %q (found %v), want %q", stored, ok, content)
	}

	digest := sha256.Sum256([]byte(content))
	if repository.RecordedLoteoFile.Sha256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("recorded sha256 = %q, want %q", repository.RecordedLoteoFile.Sha256, hex.EncodeToString(digest[:]))
	}
}

func TestStoreLoteoFileRejectsRolesOtherThanAdminOrAgrimensor(t *testing.T) {
	for _, rol := range []string{domain.RolAdministrativo, domain.RolEscribano, domain.RolInmobiliaria} {
		t.Run(rol, func(t *testing.T) {
			repository := &gatewayfake.LoteoRepository{Exists: true}
			storage := &gatewayfake.ObjectStorage{}
			useCase := loteos.NewStoreLoteoFile(repository, storage)

			actor := loteos.Actor{AuthProviderID: "actor-1", Roles: []string{rol}}
			_, err := useCase.Execute(
				context.Background(), actor, loteoFileInput(archivoLoteoID, "foto", "image/jpeg", "x"),
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

func TestStoreLoteoFileRejectsAnUnassignedAgrimensor(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, Assigned: false}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoFile(repository, storage)

	_, err := useCase.Execute(
		context.Background(), agrimensor(), loteoFileInput(archivoLoteoID, "foto", "image/jpeg", "x"),
	)

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
}

func TestStoreLoteoFileRejectsAnUnknownCategory(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoFile(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(), loteoFileInput(archivoLoteoID, "dxf", "image/jpeg", "x"),
	)

	if !errors.Is(err, domain.ErrInvalidFile) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidFile)
	}
	if storage.PutCalls != 0 {
		t.Fatalf("PutCalls = %d, want 0", storage.PutCalls)
	}
}

func TestStoreLoteoFileRejectsAnUnsupportedMimeType(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoFile(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(), loteoFileInput(archivoLoteoID, "foto", "application/zip", "x"),
	)

	if !errors.Is(err, domain.ErrInvalidFile) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidFile)
	}
}

func TestStoreLoteoFileRejectsContentThatDoesNotMatchTheDeclaredMimeType(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoFile(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(),
		loteoFileInput(archivoLoteoID, "foto", "image/jpeg", "this is plain text, not a jpeg"),
	)

	if !errors.Is(err, domain.ErrInvalidFile) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidFile)
	}
	if storage.PutCalls != 0 {
		t.Fatalf("PutCalls = %d, want 0", storage.PutCalls)
	}
}

func TestStoreLoteoFileAcceptsEachSupportedMimeTypeWithMatchingContent(t *testing.T) {
	cases := map[string]string{
		"image/jpeg":      "\xFF\xD8\xFFrest-of-the-jpeg",
		"image/png":       "\x89PNG\r\n\x1a\nrest-of-the-png",
		"image/webp":      "RIFF\x00\x00\x00\x00WEBPrest",
		"application/pdf": "%PDF-1.7 rest of the pdf",
	}
	for mimeType, content := range cases {
		t.Run(mimeType, func(t *testing.T) {
			repository := &gatewayfake.LoteoRepository{Exists: true}
			storage := &gatewayfake.ObjectStorage{}
			useCase := loteos.NewStoreLoteoFile(repository, storage)

			_, err := useCase.Execute(
				context.Background(), administrador(), loteoFileInput(archivoLoteoID, "foto", mimeType, content),
			)

			if err != nil {
				t.Fatalf("Execute() error = %v, want nil", err)
			}
			if storage.PutCalls != 1 {
				t.Fatalf("PutCalls = %d, want 1", storage.PutCalls)
			}
		})
	}
}

func TestStoreLoteoFileRejectsAnOversizedFile(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoFile(repository, storage)

	input := loteoFileInput(archivoLoteoID, "foto", "image/jpeg", "x")
	input.Size = domain.MaxFileBytes + 1

	_, err := useCase.Execute(context.Background(), administrador(), input)

	if !errors.Is(err, domain.ErrInvalidFile) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidFile)
	}
	if storage.PutCalls != 0 {
		t.Fatalf("PutCalls = %d, want 0", storage.PutCalls)
	}
}

func TestStoreLoteoFileRejectsAFullEntity(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, RecordLoteoFileErr: domain.ErrTooManyFiles}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoFile(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(), loteoFileInput(archivoLoteoID, "foto", "image/jpeg", "\xFF\xD8\xFFx"),
	)

	if !errors.Is(err, domain.ErrTooManyFiles) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrTooManyFiles)
	}
	// The quota is enforced atomically at record time (see RecordLoteoFile),
	// so the object is already in storage by the time the limit is hit and
	// must be compensated instead of never having been written.
	if storage.DeleteCalls != 1 {
		t.Fatalf("DeleteCalls = %d, want 1 (the orphaned upload is cleaned up)", storage.DeleteCalls)
	}
}

func TestStoreLoteoFileCleansUpTheUploadWhenRecordingFails(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{
		Exists:             true,
		RecordLoteoFileErr: errors.New("connection reset"),
	}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoFile(repository, storage)

	_, err := useCase.Execute(
		context.Background(), administrador(), loteoFileInput(archivoLoteoID, "foto", "image/jpeg", "\xFF\xD8\xFFx"),
	)

	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != domain.KindUnavailable {
		t.Fatalf("Execute() error = %v, want an unavailable-kind domain error", err)
	}
	if storage.DeleteCalls != 1 {
		t.Fatalf("DeleteCalls = %d, want 1", storage.DeleteCalls)
	}
}
