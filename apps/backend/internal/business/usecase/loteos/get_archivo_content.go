package loteos

import (
	"context"
	"io"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ArchivoContent is a foto/plano's own bytes, ready to be streamed back to
// the caller. Body is the object's raw stream; the caller owns it and must
// close it.
type ArchivoContent struct {
	Archivo domain.Archivo
	Body    io.ReadCloser
}

// GetArchivoContent reads a foto/plano's bytes back from storage. It applies
// the same visibility rules as GetLoteo: a loteo the actor may not see is
// reported as domain.ErrLoteoNotFound.
type GetArchivoContent interface {
	Execute(ctx context.Context, actor Actor, loteoID, archivoID string) (ArchivoContent, error)
}

type getArchivoContentUseCase struct {
	repository gateway.LoteoRepository
	storage    gateway.ObjectStorage
}

func NewGetArchivoContent(repository gateway.LoteoRepository, storage gateway.ObjectStorage) GetArchivoContent {
	return &getArchivoContentUseCase{repository: repository, storage: storage}
}

func (useCase *getArchivoContentUseCase) Execute(
	ctx context.Context,
	actor Actor,
	loteoID, archivoID string,
) (ArchivoContent, error) {
	scope, err := loteoVisibility(actor)
	if err != nil {
		return ArchivoContent{}, err
	}

	loteoID = strings.TrimSpace(loteoID)
	if loteoID == "" {
		return ArchivoContent{}, domain.ErrLoteoNotFound
	}

	if _, err := useCase.repository.Get(ctx, loteoID, scope); err != nil {
		return ArchivoContent{}, fromRepository(err)
	}

	archivo, err := useCase.repository.GetArchivo(ctx, loteoID, archivoID)
	if err != nil {
		return ArchivoContent{}, fromRepository(err)
	}

	object, err := useCase.storage.Get(ctx, archivo.StorageKey)
	if err != nil {
		return ArchivoContent{}, fromStorage(err)
	}

	return ArchivoContent{Archivo: archivo, Body: object.Body}, nil
}
