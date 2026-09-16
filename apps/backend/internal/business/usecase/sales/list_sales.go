package sales

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type ListSalesInput struct {
	Actor  Actor
	Filter domain.SaleListFilter
}

type ListSales interface {
	Execute(ctx context.Context, input ListSalesInput) (domain.SalePage, error)
}

type listSalesUseCase struct {
	repository gateway.SaleRepository
}

func NewListSales(repository gateway.SaleRepository) ListSales {
	return &listSalesUseCase{repository: repository}
}

func (useCase *listSalesUseCase) Execute(ctx context.Context, input ListSalesInput) (domain.SalePage, error) {
	scope, err := saleScope(input.Actor)
	if err != nil {
		return domain.SalePage{}, err
	}
	filter, err := input.Filter.Normalize()
	if err != nil {
		return domain.SalePage{}, err
	}
	page, err := useCase.repository.List(ctx, filter, scope)
	if err != nil {
		return domain.SalePage{}, fromRepository(err)
	}
	return page, nil
}
