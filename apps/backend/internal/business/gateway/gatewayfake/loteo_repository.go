package gatewayfake

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// LoteoRepository is a fake gateway.LoteoRepository for tests.
type LoteoRepository struct {
	CreateCalls   int
	CreateErr     error
	Created       domain.Loteo
	ReceivedLoteo domain.NewLoteo

	UpdateLoteCalls  int
	UpdateLoteErr    error
	UpdatedLote      domain.Lote
	ReceivedLoteData domain.LoteData
	ReceivedLoteoID  string
	ReceivedLoteID   string

	UpdateManzanaCalls  int
	UpdateManzanaErr    error
	UpdatedManzana      domain.Manzana
	ReceivedManzanaData domain.ManzanaData
	ReceivedManzanaID   string

	UpdateCalleCalls  int
	UpdateCalleErr    error
	UpdatedCalle      domain.Calle
	ReceivedCalleData domain.CalleData
	ReceivedCalleID   string

	Assigned     bool
	AssignedErr  error
	AssignedCall int

	Exists      bool
	ExistsErr   error
	ExistsCalls int

	RecordDxfFileCalls    int
	RecordDxfFileErr      error
	RecordedDxfLoteoID    string
	RecordedDxfFile       domain.NewLoteoDxfFile
	RecordedDxfFileResult domain.LoteoDxfFile

	RecordLoteoFileCalls     int
	RecordLoteoFileErr       error
	RecordedLoteoFileLoteoID string
	RecordedLoteoFile        domain.NewFile
	RecordedLoteoFileResult  domain.File

	RecordLoteFileCalls     int
	RecordLoteFileErr       error
	RecordedLoteFileLoteoID string
	RecordedLoteFileLoteID  string
	RecordedLoteFile        domain.NewFile
	RecordedLoteFileResult  domain.File

	ListLoteoFilesCalls   int
	ListLoteoFilesErr     error
	ListLoteoFilesLoteoID string
	ListLoteoFilesResult  []domain.File

	ListLoteFilesCalls   int
	ListLoteFilesErr     error
	ListLoteFilesLoteoID string
	ListLoteFilesLoteID  string
	ListLoteFilesResult  []domain.File

	GetFileCalls   int
	GetFileErr     error
	GetFileLoteoID string
	GetFileID      string
	GetFileResult  domain.File

	DeleteFileCalls    int
	DeleteFileErr      error
	DeletedFileLoteoID string
	DeletedFileID      string

	ListCalls  int
	ListErr    error
	ListResult []domain.LoteoSummary
	ListSearch string
	ListScope  gateway.LoteoScope

	GetCalls   int
	GetErr     error
	GetResult  domain.Loteo
	GetLoteoID string
	GetScope   gateway.LoteoScope

	ActorAuthProviderID string
}

func (fake *LoteoRepository) Create(
	_ context.Context,
	actorAuthProviderID string,
	loteo domain.NewLoteo,
) (domain.Loteo, error) {
	fake.CreateCalls++
	fake.ActorAuthProviderID = actorAuthProviderID
	fake.ReceivedLoteo = loteo
	if fake.CreateErr != nil {
		return domain.Loteo{}, fake.CreateErr
	}
	if fake.Created.ID == "" {
		return domain.Loteo{ID: "loteo-1", Name: loteo.Name}, nil
	}

	return fake.Created, nil
}

func (fake *LoteoRepository) UpdateLote(
	_ context.Context,
	actorAuthProviderID, loteoID, loteID string,
	data domain.LoteData,
) (domain.Lote, error) {
	fake.UpdateLoteCalls++
	fake.ActorAuthProviderID = actorAuthProviderID
	fake.ReceivedLoteoID = loteoID
	fake.ReceivedLoteID = loteID
	fake.ReceivedLoteData = data
	if fake.UpdateLoteErr != nil {
		return domain.Lote{}, fake.UpdateLoteErr
	}
	if fake.UpdatedLote.ID == "" {
		return domain.Lote{ID: loteID, Number: data.Number}, nil
	}

	return fake.UpdatedLote, nil
}

func (fake *LoteoRepository) UpdateManzana(
	_ context.Context,
	actorAuthProviderID, loteoID, manzanaID string,
	data domain.ManzanaData,
) (domain.Manzana, error) {
	fake.UpdateManzanaCalls++
	fake.ActorAuthProviderID = actorAuthProviderID
	fake.ReceivedLoteoID = loteoID
	fake.ReceivedManzanaID = manzanaID
	fake.ReceivedManzanaData = data
	if fake.UpdateManzanaErr != nil {
		return domain.Manzana{}, fake.UpdateManzanaErr
	}
	if fake.UpdatedManzana.ID == "" {
		return domain.Manzana{ID: manzanaID, Number: data.Number, CalleIDs: data.CalleIDs}, nil
	}

	return fake.UpdatedManzana, nil
}

func (fake *LoteoRepository) UpdateCalle(
	_ context.Context,
	actorAuthProviderID, loteoID, calleID string,
	data domain.CalleData,
) (domain.Calle, error) {
	fake.UpdateCalleCalls++
	fake.ActorAuthProviderID = actorAuthProviderID
	fake.ReceivedLoteoID = loteoID
	fake.ReceivedCalleID = calleID
	fake.ReceivedCalleData = data
	if fake.UpdateCalleErr != nil {
		return domain.Calle{}, fake.UpdateCalleErr
	}
	if fake.UpdatedCalle.ID == "" {
		return domain.Calle{ID: calleID, Name: data.Name, Type: data.Type}, nil
	}

	return fake.UpdatedCalle, nil
}

func (fake *LoteoRepository) List(
	_ context.Context,
	search string,
	scope gateway.LoteoScope,
) ([]domain.LoteoSummary, error) {
	fake.ListCalls++
	fake.ListSearch = search
	fake.ListScope = scope
	if fake.ListErr != nil {
		return nil, fake.ListErr
	}

	return fake.ListResult, nil
}

func (fake *LoteoRepository) Get(
	_ context.Context,
	loteoID string,
	scope gateway.LoteoScope,
) (domain.Loteo, error) {
	fake.GetCalls++
	fake.GetLoteoID = loteoID
	fake.GetScope = scope
	if fake.GetErr != nil {
		return domain.Loteo{}, fake.GetErr
	}
	if fake.GetResult.ID == "" {
		return domain.Loteo{ID: loteoID}, nil
	}

	return fake.GetResult, nil
}

func (fake *LoteoRepository) IsAssignedToLoteo(context.Context, string, string) (bool, error) {
	fake.AssignedCall++
	if fake.AssignedErr != nil {
		return false, fake.AssignedErr
	}

	return fake.Assigned, nil
}

func (fake *LoteoRepository) LoteoExists(context.Context, string) (bool, error) {
	fake.ExistsCalls++
	if fake.ExistsErr != nil {
		return false, fake.ExistsErr
	}

	return fake.Exists, nil
}

func (fake *LoteoRepository) RecordDxfFile(
	_ context.Context,
	actorAuthProviderID, loteoID string,
	file domain.NewLoteoDxfFile,
) (domain.LoteoDxfFile, error) {
	fake.RecordDxfFileCalls++
	fake.ActorAuthProviderID = actorAuthProviderID
	fake.RecordedDxfLoteoID = loteoID
	fake.RecordedDxfFile = file
	if fake.RecordDxfFileErr != nil {
		return domain.LoteoDxfFile{}, fake.RecordDxfFileErr
	}
	if fake.RecordedDxfFileResult.ID == "" {
		return domain.LoteoDxfFile{
			ID:           "archivo-1",
			StorageKey:   file.StorageKey,
			OriginalName: file.OriginalName,
			MimeType:     file.MimeType,
			Sha256:       file.Sha256,
		}, nil
	}

	return fake.RecordedDxfFileResult, nil
}

func (fake *LoteoRepository) RecordLoteoFile(
	_ context.Context,
	actorAuthProviderID, loteoID string,
	file domain.NewFile,
) (domain.File, error) {
	fake.RecordLoteoFileCalls++
	fake.ActorAuthProviderID = actorAuthProviderID
	fake.RecordedLoteoFileLoteoID = loteoID
	fake.RecordedLoteoFile = file
	if fake.RecordLoteoFileErr != nil {
		return domain.File{}, fake.RecordLoteoFileErr
	}
	if fake.RecordedLoteoFileResult.ID == "" {
		return domain.File{
			ID:           "archivo-loteo-1",
			Category:     file.Category,
			StorageKey:   file.StorageKey,
			OriginalName: file.OriginalName,
			MimeType:     file.MimeType,
			Sha256:       file.Sha256,
		}, nil
	}

	return fake.RecordedLoteoFileResult, nil
}

func (fake *LoteoRepository) RecordLoteFile(
	_ context.Context,
	actorAuthProviderID, loteoID, loteID string,
	file domain.NewFile,
) (domain.File, error) {
	fake.RecordLoteFileCalls++
	fake.ActorAuthProviderID = actorAuthProviderID
	fake.RecordedLoteFileLoteoID = loteoID
	fake.RecordedLoteFileLoteID = loteID
	fake.RecordedLoteFile = file
	if fake.RecordLoteFileErr != nil {
		return domain.File{}, fake.RecordLoteFileErr
	}
	if fake.RecordedLoteFileResult.ID == "" {
		return domain.File{
			ID:           "archivo-lote-1",
			Category:     file.Category,
			StorageKey:   file.StorageKey,
			OriginalName: file.OriginalName,
			MimeType:     file.MimeType,
			Sha256:       file.Sha256,
		}, nil
	}

	return fake.RecordedLoteFileResult, nil
}

func (fake *LoteoRepository) ListLoteoFiles(_ context.Context, loteoID string) ([]domain.File, error) {
	fake.ListLoteoFilesCalls++
	fake.ListLoteoFilesLoteoID = loteoID
	if fake.ListLoteoFilesErr != nil {
		return nil, fake.ListLoteoFilesErr
	}

	return fake.ListLoteoFilesResult, nil
}

func (fake *LoteoRepository) ListLoteFiles(_ context.Context, loteoID, loteID string) ([]domain.File, error) {
	fake.ListLoteFilesCalls++
	fake.ListLoteFilesLoteoID = loteoID
	fake.ListLoteFilesLoteID = loteID
	if fake.ListLoteFilesErr != nil {
		return nil, fake.ListLoteFilesErr
	}

	return fake.ListLoteFilesResult, nil
}

func (fake *LoteoRepository) GetFile(_ context.Context, loteoID, archivoID string) (domain.File, error) {
	fake.GetFileCalls++
	fake.GetFileLoteoID = loteoID
	fake.GetFileID = archivoID
	if fake.GetFileErr != nil {
		return domain.File{}, fake.GetFileErr
	}

	return fake.GetFileResult, nil
}

func (fake *LoteoRepository) DeleteFile(_ context.Context, actorAuthProviderID, loteoID, archivoID string) error {
	fake.DeleteFileCalls++
	fake.ActorAuthProviderID = actorAuthProviderID
	fake.DeletedFileLoteoID = loteoID
	fake.DeletedFileID = archivoID
	return fake.DeleteFileErr
}
