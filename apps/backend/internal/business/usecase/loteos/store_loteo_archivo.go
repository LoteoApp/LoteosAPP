package loteos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// StoreLoteoFileInput is a foto/plano attached to the loteo itself, as it
// reaches the use case. Content is already materialized (a file or a
// buffer), so it can be hashed and then rewound for the upload.
type StoreLoteoFileInput struct {
	LoteoID  string
	Category string
	FileName string
	MimeType string
	Content  io.ReadSeeker
	Size     int64
}

// StoreLoteoFile attaches a new foto/plano to a loteo. Unlike
// StoreLoteoDxf it never supersedes a prior upload: a loteo may carry
// several active fotos/planos. Only an administrador, or an agrimensor
// assigned to the loteo, may do this.
type StoreLoteoFile interface {
	Execute(ctx context.Context, actor Actor, input StoreLoteoFileInput) (domain.File, error)
}

type storeLoteoFileUseCase struct {
	repository gateway.LoteoRepository
	storage    gateway.ObjectStorage
}

func NewStoreLoteoFile(repository gateway.LoteoRepository, storage gateway.ObjectStorage) StoreLoteoFile {
	return &storeLoteoFileUseCase{repository: repository, storage: storage}
}

func (useCase *storeLoteoFileUseCase) Execute(
	ctx context.Context,
	actor Actor,
	input StoreLoteoFileInput,
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

	key, err := newFileStorageKey("loteos/" + input.LoteoID)
	if err != nil {
		return domain.File{}, domain.ErrStorageUnavailable.WithCause(err)
	}
	if err := useCase.storage.Put(ctx, key, input.Content, input.Size, input.MimeType); err != nil {
		return domain.File{}, fromStorage(err)
	}

	file, err := useCase.repository.RecordLoteoFile(ctx, actor.AuthProviderID, input.LoteoID, domain.NewFile{
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

func validateFileUpload(category, mimeType string, content io.ReadSeeker, size int64) error {
	if !domain.ValidFileCategory(category) {
		return domain.ErrInvalidFile
	}
	if content == nil || size <= 0 || size > domain.MaxFileBytes {
		return domain.ErrInvalidFile
	}
	if !domain.ValidFileMimeType(mimeType) {
		return domain.ErrInvalidFile
	}
	if err := verifyFileContentSignature(mimeType, content); err != nil {
		return err
	}

	return nil
}

// fileSignatureBytes is enough to hold the longest magic number this
// package checks for (the WEBP RIFF/WEBP pair, at 12 bytes).
const fileSignatureBytes = 12

// verifyFileContentSignature rejects an upload whose own bytes don't
// match the file type it declares in mimeType — a client can claim any
// Content-Type on the multipart part regardless of what it actually sends.
func verifyFileContentSignature(mimeType string, content io.ReadSeeker) error {
	if _, err := content.Seek(0, io.SeekStart); err != nil {
		return domain.ErrInvalidFile.WithCause(err)
	}
	header := make([]byte, fileSignatureBytes)
	n, err := io.ReadFull(content, header)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return domain.ErrInvalidFile.WithCause(err)
	}
	if _, err := content.Seek(0, io.SeekStart); err != nil {
		return domain.ErrInvalidFile.WithCause(err)
	}

	if !domain.FileContentMatchesMimeType(mimeType, header[:n]) {
		return domain.ErrInvalidFile
	}

	return nil
}

func newFileStorageKey(prefix string) (string, error) {
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
