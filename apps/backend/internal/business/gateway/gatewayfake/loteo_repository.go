package gatewayfake

import (
	"context"

	"loteosapp/backend/internal/business/domain"
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

	Assigned     bool
	AssignedErr  error
	AssignedCall int

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

func (fake *LoteoRepository) IsAssignedToLoteo(context.Context, string, string) (bool, error) {
	fake.AssignedCall++
	if fake.AssignedErr != nil {
		return false, fake.AssignedErr
	}

	return fake.Assigned, nil
}
