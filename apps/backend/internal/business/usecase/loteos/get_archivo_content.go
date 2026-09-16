package loteos

import (
	"context"
	"io"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// FileContent is a foto/plano's own bytes, ready to be streamed back to
// the caller. Body is the object's raw stream; the caller owns it and must
// close it. Size is the exact byte length of Body, so the caller can declare
// Content-Length and let the transport detect a stream that ends early.
type FileContent struct {
	File domain.File
	Body io.ReadCloser
	Size int64
}

// GetFileContent reads a foto/plano's bytes back from storage. It applies
// the same visibility rules as GetLoteo: a loteo the actor may not see is
// reported as domain.ErrLoteoNotFound.
type GetFileContent interface {
	Execute(ctx context.Context, actor Actor, loteoID, fileID string) (FileContent, error)
}

type getFileContentUseCase struct {
	repository gateway.LoteoRepository
	storage    gateway.ObjectStorage
}

func NewGetFileContent(repository gateway.LoteoRepository, storage gateway.ObjectStorage) GetFileContent {
	return &getFileContentUseCase{repository: repository, storage: storage}
}

func (useCase *getFileContentUseCase) Execute(
	ctx context.Context,
	actor Actor,
	loteoID, fileID string,
) (FileContent, error) {
	scope, err := loteoVisibility(actor)
	if err != nil {
		return FileContent{}, err
	}

	loteoID = strings.TrimSpace(loteoID)
	if loteoID == "" {
		return FileContent{}, domain.ErrLoteoNotFound
	}

	if _, err := useCase.repository.Get(ctx, loteoID, scope); err != nil {
		return FileContent{}, fromRepository(err)
	}

	file, err := useCase.repository.GetFile(ctx, loteoID, fileID)
	if err != nil {
		return FileContent{}, fromRepository(err)
	}

	object, err := useCase.storage.Get(ctx, file.StorageKey)
	if err != nil {
		return FileContent{}, fromStorage(err)
	}

	return FileContent{File: file, Body: object.Body, Size: object.Size}, nil
}
