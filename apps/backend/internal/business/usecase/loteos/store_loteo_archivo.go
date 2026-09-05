package loteos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// StoreLoteoArchivoInput is a foto/plano attached to the loteo itself, as it
// reaches the use case. Content is already materialized (a file or a
// buffer), so it can be hashed and then rewound for the upload.
type StoreLoteoArchivoInput struct {
	LoteoID   string
	Categoria string
	FileName  string
	MimeType  string
	Content   io.ReadSeeker
	Size      int64
}

// StoreLoteoArchivo attaches a new foto/plano to a loteo. Unlike
// StoreLoteoDxf it never supersedes a prior upload: a loteo may carry
// several active fotos/planos. Only an administrador, or an agrimensor
// assigned to the loteo, may do this.
type StoreLoteoArchivo interface {
	Execute(ctx context.Context, actor Actor, input StoreLoteoArchivoInput) (domain.Archivo, error)
}

type storeLoteoArchivoUseCase struct {
	repository gateway.LoteoRepository
	storage    gateway.ObjectStorage
}

func NewStoreLoteoArchivo(repository gateway.LoteoRepository, storage gateway.ObjectStorage) StoreLoteoArchivo {
	return &storeLoteoArchivoUseCase{repository: repository, storage: storage}
}

func (useCase *storeLoteoArchivoUseCase) Execute(
	ctx context.Context,
	actor Actor,
	input StoreLoteoArchivoInput,
) (domain.Archivo, error) {
	if err := authorizeEditor(ctx, useCase.repository, actor, input.LoteoID); err != nil {
		return domain.Archivo{}, err
	}

	if err := validateArchivoUpload(input.Categoria, input.MimeType, input.Content, input.Size); err != nil {
		return domain.Archivo{}, err
	}

	existing, err := useCase.repository.ListLoteoArchivos(ctx, input.LoteoID)
	if err != nil {
		return domain.Archivo{}, fromRepository(err)
	}
	if len(existing) >= domain.MaxArchivosPerEntity {
		return domain.Archivo{}, domain.ErrTooManyArchivos
	}

	digest, err := hashAndRewind(input.Content)
	if err != nil {
		return domain.Archivo{}, domain.ErrInvalidArchivo.WithCause(err)
	}

	key, err := newArchivoStorageKey("loteos/" + input.LoteoID)
	if err != nil {
		return domain.Archivo{}, domain.ErrStorageUnavailable.WithCause(err)
	}
	if err := useCase.storage.Put(ctx, key, input.Content, input.Size, input.MimeType); err != nil {
		return domain.Archivo{}, fromStorage(err)
	}

	archivo, err := useCase.repository.RecordLoteoArchivo(ctx, actor.AuthProviderID, input.LoteoID, domain.NewArchivo{
		Categoria:    input.Categoria,
		StorageKey:   key,
		OriginalName: strings.TrimSpace(input.FileName),
		MimeType:     input.MimeType,
		Sha256:       digest,
	})
	if err != nil {
		return domain.Archivo{}, cleanupAfterRecordFailure(ctx, useCase.storage, key, err)
	}

	return archivo, nil
}

func validateArchivoUpload(categoria, mimeType string, content io.ReadSeeker, size int64) error {
	if !domain.ValidArchivoCategoria(categoria) {
		return domain.ErrInvalidArchivo
	}
	if content == nil || size <= 0 || size > domain.MaxArchivoFileBytes {
		return domain.ErrInvalidArchivo
	}
	if !domain.ValidArchivoMimeType(mimeType) {
		return domain.ErrInvalidArchivo
	}

	return nil
}

func newArchivoStorageKey(prefix string) (string, error) {
	var suffix [16]byte
	if _, err := io.ReadFull(rand.Reader, suffix[:]); err != nil {
		return "", err
	}

	return prefix + "/archivos/" + hex.EncodeToString(suffix[:]), nil
}

// cleanupAfterRecordFailure deletes the object just written to storage once
// the database write that would have recorded it fails, the same way
// StoreLoteoDxf does. dxfCleanupTimeout and withCleanupCause are shared with
// store_loteo_dxf.go: both use the same object storage and the same
// compensating-delete shape.
func cleanupAfterRecordFailure(ctx context.Context, storage gateway.ObjectStorage, key string, recordErr error) error {
	mapped := fromRepository(recordErr)
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), dxfCleanupTimeout)
	defer cancel()

	if cleanupErr := storage.Delete(cleanupCtx, key); cleanupErr != nil {
		return withCleanupCause(mapped, recordErr, key, cleanupErr)
	}

	return mapped
}
