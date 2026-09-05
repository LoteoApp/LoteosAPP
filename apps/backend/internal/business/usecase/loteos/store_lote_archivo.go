package loteos

import (
	"context"
	"io"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// StoreLoteArchivoInput is a foto/plano attached to one lote, as it reaches
// the use case. Content is already materialized (a file or a buffer), so it
// can be hashed and then rewound for the upload.
type StoreLoteArchivoInput struct {
	LoteoID   string
	LoteID    string
	Categoria string
	FileName  string
	MimeType  string
	Content   io.ReadSeeker
	Size      int64
}

// StoreLoteArchivo attaches a new foto/plano to one lote. A lote may carry
// several active fotos/planos. Only an administrador, or an agrimensor
// assigned to the loteo, may do this.
type StoreLoteArchivo interface {
	Execute(ctx context.Context, actor Actor, input StoreLoteArchivoInput) (domain.Archivo, error)
}

type storeLoteArchivoUseCase struct {
	repository gateway.LoteoRepository
	storage    gateway.ObjectStorage
}

func NewStoreLoteArchivo(repository gateway.LoteoRepository, storage gateway.ObjectStorage) StoreLoteArchivo {
	return &storeLoteArchivoUseCase{repository: repository, storage: storage}
}

func (useCase *storeLoteArchivoUseCase) Execute(
	ctx context.Context,
	actor Actor,
	input StoreLoteArchivoInput,
) (domain.Archivo, error) {
	if err := authorizeEditor(ctx, useCase.repository, actor, input.LoteoID); err != nil {
		return domain.Archivo{}, err
	}

	if err := validateArchivoUpload(input.Categoria, input.MimeType, input.Content, input.Size); err != nil {
		return domain.Archivo{}, err
	}

	existing, err := useCase.repository.ListLoteArchivos(ctx, input.LoteoID, input.LoteID)
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

	key, err := newArchivoStorageKey("loteos/" + input.LoteoID + "/lotes/" + input.LoteID)
	if err != nil {
		return domain.Archivo{}, domain.ErrStorageUnavailable.WithCause(err)
	}
	if err := useCase.storage.Put(ctx, key, input.Content, input.Size, input.MimeType); err != nil {
		return domain.Archivo{}, fromStorage(err)
	}

	archivo, err := useCase.repository.RecordLoteArchivo(ctx, actor.AuthProviderID, input.LoteoID, input.LoteID, domain.NewArchivo{
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
