package loteos

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// StoreLoteoDxfInput is the original DXF of a loteo as it reaches the use
// case. Content is already materialized (a file or a buffer), so it can be
// hashed and then rewound for the upload.
type StoreLoteoDxfInput struct {
	LoteoID  string
	FileName string
	Content  io.ReadSeeker
	Size     int64
}

const (
	dxfMimeType       = "application/dxf"
	dxfProbeBytes     = 4 << 10
	dxfCleanupTimeout = 5 * time.Second
)

// StoreLoteoDxf stores the original DXF of an existing loteo in object
// storage and records it in the archivos table. Only an administrador, or an
// agrimensor assigned to the loteo, may do this.
type StoreLoteoDxf interface {
	Execute(ctx context.Context, actor Actor, input StoreLoteoDxfInput) (domain.LoteoDxfFile, error)
}

type storeLoteoDxfUseCase struct {
	repository gateway.LoteoRepository
	storage    gateway.ObjectStorage
}

func NewStoreLoteoDxf(repository gateway.LoteoRepository, storage gateway.ObjectStorage) StoreLoteoDxf {
	return &storeLoteoDxfUseCase{repository: repository, storage: storage}
}

// Execute uploads under a new key before writing the archivos row. If that
// write fails, only the unrecorded version is eligible for cleanup.
func (useCase *storeLoteoDxfUseCase) Execute(
	ctx context.Context,
	actor Actor,
	input StoreLoteoDxfInput,
) (domain.LoteoDxfFile, error) {
	isAdmin := domain.HasRole(actor.Roles, domain.RolAdministrador)
	if !isAdmin && !domain.HasRole(actor.Roles, domain.RolAgrimensor) {
		return domain.LoteoDxfFile{}, domain.ErrNoAutorizado
	}

	if !isAdmin {
		assigned, err := useCase.repository.IsAssignedToLoteo(ctx, actor.AuthProviderID, input.LoteoID)
		if err != nil {
			return domain.LoteoDxfFile{}, fromRepository(err)
		}
		if !assigned {
			return domain.LoteoDxfFile{}, domain.ErrNoAutorizado
		}
	}

	if input.Content == nil || input.Size <= 0 || input.Size > domain.MaxDxfFileBytes || !hasDxfExtension(input.FileName) {
		return domain.LoteoDxfFile{}, domain.ErrInvalidDxfFile
	}
	if err := validateDxfEnvelope(input.Content, input.Size); err != nil {
		return domain.LoteoDxfFile{}, domain.ErrInvalidDxfFile.WithCause(err)
	}

	exists, err := useCase.repository.LoteoExists(ctx, input.LoteoID)
	if err != nil {
		return domain.LoteoDxfFile{}, fromRepository(err)
	}
	if !exists {
		return domain.LoteoDxfFile{}, domain.ErrLoteoNotFound
	}

	digest, err := hashAndRewind(input.Content)
	if err != nil {
		return domain.LoteoDxfFile{}, domain.ErrInvalidDxfFile.WithCause(err)
	}

	key, err := newDxfStorageKey(input.LoteoID)
	if err != nil {
		return domain.LoteoDxfFile{}, domain.ErrStorageUnavailable.WithCause(err)
	}
	if err := useCase.storage.Put(ctx, key, input.Content, input.Size, dxfMimeType); err != nil {
		return domain.LoteoDxfFile{}, err
	}

	file, err := useCase.repository.RecordDxfFile(ctx, actor.AuthProviderID, input.LoteoID, domain.NewLoteoDxfFile{
		StorageKey:   key,
		OriginalName: strings.TrimSpace(input.FileName),
		MimeType:     dxfMimeType,
		Sha256:       digest,
	})
	if err != nil {
		mapped := fromRepository(err)
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), dxfCleanupTimeout)
		cleanupErr := useCase.storage.Delete(cleanupCtx, key)
		cancel()
		if cleanupErr != nil {
			return domain.LoteoDxfFile{}, withCleanupCause(mapped, err, key, cleanupErr)
		}
		return domain.LoteoDxfFile{}, mapped
	}

	return file, nil
}

func hasDxfExtension(name string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(name)), ".dxf")
}

func validateDxfEnvelope(content io.ReadSeeker, size int64) error {
	probeSize := min(size, dxfProbeBytes)
	head := make([]byte, probeSize)
	if _, err := content.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err := io.ReadFull(content, head); err != nil {
		return err
	}
	headFields := strings.Fields(string(head))
	if !containsTokenPair(headFields, "0", "SECTION") {
		return errors.New("DXF header is missing")
	}

	tailStart := max(int64(0), size-dxfProbeBytes)
	if _, err := content.Seek(tailStart, io.SeekStart); err != nil {
		return err
	}
	tail, err := io.ReadAll(io.LimitReader(content, dxfProbeBytes))
	if err != nil {
		return err
	}
	tailFields := strings.Fields(string(tail))
	if len(tailFields) == 0 || tailFields[len(tailFields)-1] != "EOF" {
		return errors.New("DXF terminator is missing")
	}

	_, err = content.Seek(0, io.SeekStart)
	return err
}

func containsTokenPair(fields []string, first, second string) bool {
	for index := 0; index+1 < len(fields); index++ {
		if fields[index] == first && fields[index+1] == second {
			return true
		}
	}

	return false
}

func newDxfStorageKey(loteoID string) (string, error) {
	var suffix [16]byte
	if _, err := io.ReadFull(rand.Reader, suffix[:]); err != nil {
		return "", err
	}

	return "loteos/" + loteoID + "/dxf/" + hex.EncodeToString(suffix[:]) + ".dxf", nil
}

func withCleanupCause(mapped, repositoryErr error, key string, cleanupErr error) error {
	var domainErr *domain.Error
	if !errors.As(mapped, &domainErr) {
		return mapped
	}

	cause := repositoryErr
	if domainErr.Cause != nil {
		cause = domainErr.Cause
	}
	cleanupFailure := fmt.Errorf("delete uploaded DXF %q: %w", key, cleanupErr)
	return domainErr.WithCause(errors.Join(cause, cleanupFailure))
}

func hashAndRewind(content io.ReadSeeker) (string, error) {
	hasher := sha256.New()
	if _, err := content.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	if _, err := io.Copy(hasher, content); err != nil {
		return "", err
	}
	if _, err := content.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
