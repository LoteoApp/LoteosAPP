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

const dxfLoteoID = "11111111-1111-1111-1111-111111111111"
const validDxf = "0\nSECTION\n2\nENTITIES\n0\nENDSEC\n0\nEOF\n"

func agrimensor() loteos.Actor {
	return loteos.Actor{AuthProviderID: "actor-1", Roles: []string{domain.RolAgrimensor}}
}

func dxfInput(loteoID, content string) loteos.StoreLoteoDxfInput {
	return loteos.StoreLoteoDxfInput{
		LoteoID:  loteoID,
		FileName: "plano.dxf",
		Content:  bytes.NewReader([]byte(content)),
		Size:     int64(len(content)),
	}
}

func TestStoreLoteoDxfStoresBytesThenRecordsTheFile(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)

	const content = validDxf
	file, err := useCase.Execute(context.Background(), administrador(), dxfInput(dxfLoteoID, content))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantPrefix := "loteos/" + dxfLoteoID + "/dxf/"
	if !strings.HasPrefix(file.StorageKey, wantPrefix) || !strings.HasSuffix(file.StorageKey, ".dxf") {
		t.Fatalf("StorageKey = %q, want a versioned key under %q", file.StorageKey, wantPrefix)
	}
	if storage.PutCalls != 1 || repository.RecordDxfFileCalls != 1 {
		t.Fatalf("PutCalls = %d, RecordDxfFileCalls = %d, want 1 and 1", storage.PutCalls, repository.RecordDxfFileCalls)
	}

	stored, ok := storage.Contents(file.StorageKey)
	if !ok || string(stored) != content {
		t.Fatalf("stored object = %q (found %v), want %q", stored, ok, content)
	}

	digest := sha256.Sum256([]byte(content))
	if repository.RecordedDxfFile.Sha256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("recorded sha256 = %q, want %q", repository.RecordedDxfFile.Sha256, hex.EncodeToString(digest[:]))
	}
	if repository.RecordedDxfLoteoID != dxfLoteoID {
		t.Fatalf("recorded loteo id = %q, want %q", repository.RecordedDxfLoteoID, dxfLoteoID)
	}
	storedObject, err := storage.Get(context.Background(), file.StorageKey)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer storedObject.Body.Close()
	if storedObject.ContentType != "application/dxf" || repository.RecordedDxfFile.MimeType != "application/dxf" {
		t.Fatalf("content types = %q and %q, want application/dxf", storedObject.ContentType, repository.RecordedDxfFile.MimeType)
	}
}

func TestStoreLoteoDxfRejectsRolesOtherThanAdminOrAgrimensor(t *testing.T) {
	for _, rol := range []string{domain.RolAdministrativo, domain.RolEscribano, domain.RolInmobiliaria} {
		t.Run(rol, func(t *testing.T) {
			repository := &gatewayfake.LoteoRepository{Exists: true}
			storage := &gatewayfake.ObjectStorage{}
			useCase := loteos.NewStoreLoteoDxf(repository, storage)

			actor := loteos.Actor{AuthProviderID: "actor-1", Roles: []string{rol}}
			_, err := useCase.Execute(context.Background(), actor, dxfInput(dxfLoteoID, validDxf))

			if !errors.Is(err, domain.ErrNoAutorizado) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
			}
			if storage.PutCalls != 0 || repository.ExistsCalls != 0 {
				t.Fatalf("PutCalls = %d, ExistsCalls = %d, want 0 and 0", storage.PutCalls, repository.ExistsCalls)
			}
		})
	}
}

func TestStoreLoteoDxfAllowsAgrimensorAssignedToTheLoteo(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, Assigned: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)

	if _, err := useCase.Execute(context.Background(), agrimensor(), dxfInput(dxfLoteoID, validDxf)); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if storage.PutCalls != 1 {
		t.Fatalf("PutCalls = %d, want 1", storage.PutCalls)
	}
}

func TestStoreLoteoDxfSurfacesAnAssignmentLookupFailure(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, AssignedErr: errors.New("connection reset")}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)

	_, err := useCase.Execute(context.Background(), agrimensor(), dxfInput(dxfLoteoID, validDxf))
	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
	if storage.PutCalls != 0 {
		t.Fatalf("PutCalls = %d, want 0", storage.PutCalls)
	}
}

type failingReadSeeker struct{}

func (failingReadSeeker) Read([]byte) (int, error)       { return 0, errors.New("read failed") }
func (failingReadSeeker) Seek(int64, int) (int64, error) { return 0, nil }

func TestStoreLoteoDxfRejectsContentItCannotHash(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)

	input := dxfInput(dxfLoteoID, validDxf)
	input.Content = failingReadSeeker{}

	_, err := useCase.Execute(context.Background(), administrador(), input)
	if !errors.Is(err, domain.ErrInvalidDxfFile) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidDxfFile)
	}
	if storage.PutCalls != 0 {
		t.Fatalf("PutCalls = %d, want 0", storage.PutCalls)
	}
}

func TestStoreLoteoDxfRejectsAgrimensorNotAssignedToTheLoteo(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, Assigned: false}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)

	_, err := useCase.Execute(context.Background(), agrimensor(), dxfInput(dxfLoteoID, validDxf))
	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if storage.PutCalls != 0 {
		t.Fatalf("PutCalls = %d, want 0", storage.PutCalls)
	}
	if repository.ExistsCalls != 0 {
		t.Fatalf("ExistsCalls = %d, want 0 so missing and unassigned loteos are indistinguishable", repository.ExistsCalls)
	}
}

func TestStoreLoteoDxfReturnsNotFoundForUnknownLoteo(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: false}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)

	_, err := useCase.Execute(context.Background(), administrador(), dxfInput(dxfLoteoID, validDxf))
	if !errors.Is(err, domain.ErrLoteoNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrLoteoNotFound)
	}
	if storage.PutCalls != 0 {
		t.Fatalf("PutCalls = %d, want 0", storage.PutCalls)
	}
}

func TestStoreLoteoDxfRejectsInvalidFile(t *testing.T) {
	cases := map[string]loteos.StoreLoteoDxfInput{
		"empty":            {LoteoID: dxfLoteoID, FileName: "plano.dxf", Content: bytes.NewReader(nil), Size: 0},
		"too large":        {LoteoID: dxfLoteoID, FileName: "plano.dxf", Content: bytes.NewReader([]byte("x")), Size: domain.MaxDxfFileBytes + 1},
		"not a dxf":        {LoteoID: dxfLoteoID, FileName: "plano.txt", Content: bytes.NewReader([]byte("x")), Size: 1},
		"no name":          {LoteoID: dxfLoteoID, FileName: "", Content: bytes.NewReader([]byte("x")), Size: 1},
		"no content":       {LoteoID: dxfLoteoID, FileName: "plano.dxf", Content: nil, Size: 1},
		"invalid envelope": dxfInput(dxfLoteoID, "not really a dxf"),
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			repository := &gatewayfake.LoteoRepository{Exists: true}
			storage := &gatewayfake.ObjectStorage{}
			useCase := loteos.NewStoreLoteoDxf(repository, storage)

			_, err := useCase.Execute(context.Background(), administrador(), input)
			if !errors.Is(err, domain.ErrInvalidDxfFile) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidDxfFile)
			}
			if repository.ExistsCalls != 0 || storage.PutCalls != 0 {
				t.Fatalf("ExistsCalls = %d, PutCalls = %d, want 0 and 0", repository.ExistsCalls, storage.PutCalls)
			}
		})
	}
}

func TestStoreLoteoDxfReturnsStorageErrorWithoutRecording(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{PutErr: domain.ErrStorageUnavailable}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)

	_, err := useCase.Execute(context.Background(), administrador(), dxfInput(dxfLoteoID, validDxf))
	if !errors.Is(err, domain.ErrStorageUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrStorageUnavailable)
	}
	if repository.RecordDxfFileCalls != 0 {
		t.Fatalf("RecordDxfFileCalls = %d, want 0", repository.RecordDxfFileCalls)
	}
}

func TestStoreLoteoDxfClassifiesUnexpectedStorageError(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{PutErr: errors.New("connection refused")}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)

	_, err := useCase.Execute(context.Background(), administrador(), dxfInput(dxfLoteoID, validDxf))
	if !errors.Is(err, domain.ErrStorageUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrStorageUnavailable)
	}
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Cause == nil {
		t.Fatalf("Execute() error = %v, want a domain error with cause", err)
	}
	if repository.RecordDxfFileCalls != 0 {
		t.Fatalf("RecordDxfFileCalls = %d, want 0", repository.RecordDxfFileCalls)
	}
}

func TestStoreLoteoDxfDeletesObjectWhenRecordFails(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, RecordDxfFileErr: errors.New("insert failed")}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)

	_, err := useCase.Execute(context.Background(), administrador(), dxfInput(dxfLoteoID, validDxf))
	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
	if storage.DeleteCalls != 1 {
		t.Fatalf("DeleteCalls = %d, want 1", storage.DeleteCalls)
	}
	if _, ok := storage.Contents(repository.RecordedDxfFile.StorageKey); ok {
		t.Fatal("object still present after cleanup")
	}
}

func TestStoreLoteoDxfCleansUpWithAContextDetachedFromTheRequest(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, RecordDxfFileErr: errors.New("insert failed")}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := useCase.Execute(ctx, administrador(), dxfInput(dxfLoteoID, validDxf))
	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
	if storage.DeleteCalls != 1 || storage.DeleteContextErr != nil {
		t.Fatalf("DeleteCalls = %d, DeleteContextErr = %v, want one cleanup with a live context", storage.DeleteCalls, storage.DeleteContextErr)
	}
}

func TestStoreLoteoDxfReportsAFailedCleanupAsTheErrorCause(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true, RecordDxfFileErr: errors.New("insert failed")}
	storage := &gatewayfake.ObjectStorage{DeleteErr: errors.New("delete failed")}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)

	_, err := useCase.Execute(context.Background(), administrador(), dxfInput(dxfLoteoID, validDxf))
	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("Execute() error = %T, want *domain.Error", err)
	}
	if domainErr.Cause == nil || !strings.Contains(domainErr.Cause.Error(), "delete failed") {
		t.Fatalf("Execute() error cause = %v, want the cleanup failure", domainErr.Cause)
	}
}

func TestStoreLoteoDxfKeepsThePreviousObjectWhenAReplacementCannotBeRecorded(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)

	first, err := useCase.Execute(context.Background(), administrador(), dxfInput(dxfLoteoID, validDxf))
	if err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	repository.RecordDxfFileErr = errors.New("insert failed")
	_, err = useCase.Execute(context.Background(), administrador(), dxfInput(dxfLoteoID, validDxf+"\n"))
	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("replacement Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}

	if _, ok := storage.Contents(first.StorageKey); !ok {
		t.Fatal("previous object was removed after the replacement failed")
	}
	if first.StorageKey == repository.RecordedDxfFile.StorageKey {
		t.Fatal("replacement reused the previous object's storage key")
	}
	if _, ok := storage.Contents(repository.RecordedDxfFile.StorageKey); ok {
		t.Fatal("failed replacement object still exists")
	}
}

func TestStoreLoteoDxfRejectsFileNameCaseInsensitively(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)

	input := dxfInput(dxfLoteoID, validDxf)
	input.FileName = strings.ToUpper("plano.dxf")
	if _, err := useCase.Execute(context.Background(), administrador(), input); err != nil {
		t.Fatalf("Execute() with .DXF name error = %v", err)
	}
}

func TestStoreLoteoDxfAcceptsACommentBeforeTheFirstSection(t *testing.T) {
	repository := &gatewayfake.LoteoRepository{Exists: true}
	storage := &gatewayfake.ObjectStorage{}
	useCase := loteos.NewStoreLoteoDxf(repository, storage)

	content := "999\nGenerated by survey software\n" + validDxf
	if _, err := useCase.Execute(context.Background(), administrador(), dxfInput(dxfLoteoID, content)); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}
