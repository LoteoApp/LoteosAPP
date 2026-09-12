package loteos

import (
	"context"
	"io"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// StoreLoteFileInput is a foto/plano attached to one lote, as it reaches
// the use case. Content is already materialized (a file or a buffer), so it
// can be hashed and then rewound for the upload.
type StoreLoteFileInput struct {
	LoteoID  string
	LoteID   string
	Category string
	FileName string
	MimeType string
	Content  io.ReadSeeker
	Size     int64
}

// StoreLoteFile attaches a new foto/plano to one lote. A lote may carry
// several active fotos/planos. Only an administrador, or an agrimensor
// assigned to the loteo, may do this.
type StoreLoteFile interface {
	Execute(ctx context.Context, actor Actor, input StoreLoteFileInput) (domain.File, error)
}

type storeLoteFileUseCase struct {
	repository gateway.LoteoRepository
	storage    gateway.ObjectStorage
}

func NewStoreLoteFile(repository gateway.LoteoRepository, storage gateway.ObjectStorage) StoreLoteFile {
	return &storeLoteFileUseCase{repository: repository, storage: storage}
}

func (useCase *storeLoteFileUseCase) Execute(
	ctx context.Context,
	actor Actor,
	input StoreLoteFileInput,
) (domain.File, error) {
	if err := authorizeEditor(ctx, useCase.repository, actor, input.LoteoID); err != nil {
		return domain.File{}, err
	}

	if err := validateFileUpload(input.Category, input.MimeType, input.Content, input.Size); err != nil {
		return domain.File{}, err
	}

	digest, err := hashAndRewind(input.Content)
	if err != nil {
		return domain.File{}, domain.ErrInvalidFile.WithCause(err)
	}

	key, err := newFileStorageKey("loteos/" + input.LoteoID + "/lotes/" + input.LoteID)
	if err != nil {
		return domain.File{}, domain.ErrStorageUnavailable.WithCause(err)
	}
	if err := useCase.storage.Put(ctx, key, input.Content, input.Size, input.MimeType); err != nil {
		return domain.File{}, fromStorage(err)
	}

	file, err := useCase.repository.RecordLoteFile(ctx, actor.AuthProviderID, input.LoteoID, input.LoteID, domain.NewFile{
		Category:     input.Category,
		StorageKey:   key,
		OriginalName: strings.TrimSpace(input.FileName),
		MimeType:     input.MimeType,
		Sha256:       digest,
	})
	if err != nil {
		return domain.File{}, cleanupAfterRecordFailure(ctx, useCase.storage, key, err)
	}

	return file, nil
}
