package reservations

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type ListEligibleSellersInput struct {
	Actor   Actor
	LoteoID string
}

type ListEligibleSellers interface {
	Execute(ctx context.Context, input ListEligibleSellersInput) ([]domain.SellerOption, error)
}

type listEligibleSellersUseCase struct {
	repository gateway.ReservationRepository
}

func NewListEligibleSellers(repository gateway.ReservationRepository) ListEligibleSellers {
	return &listEligibleSellersUseCase{repository: repository}
}

func (useCase *listEligibleSellersUseCase) Execute(ctx context.Context, input ListEligibleSellersInput) ([]domain.SellerOption, error) {
	scope, err := reservationScope(input.Actor)
	if err != nil {
		return nil, err
	}
	loteoID := strings.TrimSpace(input.LoteoID)
	if loteoID == "" {
		return nil, domain.ErrLoteoNotFound
	}
	sellers, err := useCase.repository.ListEligibleSellers(ctx, loteoID, scope)
	if err != nil {
		return nil, fromRepository(err)
	}
	return sellers, nil
}
