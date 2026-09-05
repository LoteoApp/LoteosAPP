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

	RecordLoteoArchivoCalls     int
	RecordLoteoArchivoErr       error
	RecordedLoteoArchivoLoteoID string
	RecordedLoteoArchivo        domain.NewArchivo
	RecordedLoteoArchivoResult  domain.Archivo

	RecordLoteArchivoCalls     int
	RecordLoteArchivoErr       error
	RecordedLoteArchivoLoteoID string
	RecordedLoteArchivoLoteID  string
	RecordedLoteArchivo        domain.NewArchivo
	RecordedLoteArchivoResult  domain.Archivo

	ListLoteoArchivosCalls   int
	ListLoteoArchivosErr     error
	ListLoteoArchivosLoteoID string
	ListLoteoArchivosResult  []domain.Archivo

	ListLoteArchivosCalls   int
	ListLoteArchivosErr     error
	ListLoteArchivosLoteoID string
	ListLoteArchivosLoteID  string
	ListLoteArchivosResult  []domain.Archivo

	GetArchivoCalls   int
	GetArchivoErr     error
	GetArchivoLoteoID string
	GetArchivoID      string
	GetArchivoResult  domain.Archivo

	DeleteArchivoCalls    int
	DeleteArchivoErr      error
	DeletedArchivoLoteoID string
	DeletedArchivoID      string

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

func (fake *LoteoRepository) RecordLoteoArchivo(
	_ context.Context,
	actorAuthProviderID, loteoID string,
	file domain.NewArchivo,
) (domain.Archivo, error) {
	fake.RecordLoteoArchivoCalls++
	fake.ActorAuthProviderID = actorAuthProviderID
	fake.RecordedLoteoArchivoLoteoID = loteoID
	fake.RecordedLoteoArchivo = file
	if fake.RecordLoteoArchivoErr != nil {
		return domain.Archivo{}, fake.RecordLoteoArchivoErr
	}
	if fake.RecordedLoteoArchivoResult.ID == "" {
		return domain.Archivo{
			ID:           "archivo-loteo-1",
			Categoria:    file.Categoria,
			StorageKey:   file.StorageKey,
			OriginalName: file.OriginalName,
			MimeType:     file.MimeType,
			Sha256:       file.Sha256,
		}, nil
	}

	return fake.RecordedLoteoArchivoResult, nil
}

func (fake *LoteoRepository) RecordLoteArchivo(
	_ context.Context,
	actorAuthProviderID, loteoID, loteID string,
	file domain.NewArchivo,
) (domain.Archivo, error) {
	fake.RecordLoteArchivoCalls++
	fake.ActorAuthProviderID = actorAuthProviderID
	fake.RecordedLoteArchivoLoteoID = loteoID
	fake.RecordedLoteArchivoLoteID = loteID
	fake.RecordedLoteArchivo = file
	if fake.RecordLoteArchivoErr != nil {
		return domain.Archivo{}, fake.RecordLoteArchivoErr
	}
	if fake.RecordedLoteArchivoResult.ID == "" {
		return domain.Archivo{
			ID:           "archivo-lote-1",
			Categoria:    file.Categoria,
			StorageKey:   file.StorageKey,
			OriginalName: file.OriginalName,
			MimeType:     file.MimeType,
			Sha256:       file.Sha256,
		}, nil
	}

	return fake.RecordedLoteArchivoResult, nil
}

func (fake *LoteoRepository) ListLoteoArchivos(_ context.Context, loteoID string) ([]domain.Archivo, error) {
	fake.ListLoteoArchivosCalls++
	fake.ListLoteoArchivosLoteoID = loteoID
	if fake.ListLoteoArchivosErr != nil {
		return nil, fake.ListLoteoArchivosErr
	}

	return fake.ListLoteoArchivosResult, nil
}

func (fake *LoteoRepository) ListLoteArchivos(_ context.Context, loteoID, loteID string) ([]domain.Archivo, error) {
	fake.ListLoteArchivosCalls++
	fake.ListLoteArchivosLoteoID = loteoID
	fake.ListLoteArchivosLoteID = loteID
	if fake.ListLoteArchivosErr != nil {
		return nil, fake.ListLoteArchivosErr
	}

	return fake.ListLoteArchivosResult, nil
}

func (fake *LoteoRepository) GetArchivo(_ context.Context, loteoID, archivoID string) (domain.Archivo, error) {
	fake.GetArchivoCalls++
	fake.GetArchivoLoteoID = loteoID
	fake.GetArchivoID = archivoID
	if fake.GetArchivoErr != nil {
		return domain.Archivo{}, fake.GetArchivoErr
	}

	return fake.GetArchivoResult, nil
}

func (fake *LoteoRepository) DeleteArchivo(_ context.Context, actorAuthProviderID, loteoID, archivoID string) error {
	fake.DeleteArchivoCalls++
	fake.ActorAuthProviderID = actorAuthProviderID
	fake.DeletedArchivoLoteoID = loteoID
	fake.DeletedArchivoID = archivoID
	return fake.DeleteArchivoErr
}
