package sales

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type GetSale interface {
	Execute(ctx context.Context, actor Actor, id string) (domain.Sale, error)
}

type getSaleUseCase struct {
	repository gateway.SaleRepository
}

func NewGetSale(repository gateway.SaleRepository) GetSale {
	return &getSaleUseCase{repository: repository}
}

func (useCase *getSaleUseCase) Execute(ctx context.Context, actor Actor, id string) (domain.Sale, error) {
	scope, err := saleScope(actor)
	if err != nil {
		return domain.Sale{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Sale{}, domain.ErrSaleNotFound
	}
	sale, err := useCase.repository.Get(ctx, id, scope)
	if err != nil {
		return domain.Sale{}, fromRepository(err)
	}
	return sale, nil
}
