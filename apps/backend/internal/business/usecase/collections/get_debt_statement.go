package collections

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// GetDebtStatement reads the estado de deuda of a financed venta: entrega,
// cuotas with their state as of now, totals and the cobros registered.
type GetDebtStatement interface {
	Execute(ctx context.Context, actor Actor, saleID string) (domain.DebtStatement, error)
}

type getDebtStatementUseCase struct {
	repository gateway.CollectionRepository
	clock      Clock
}

func NewGetDebtStatement(repository gateway.CollectionRepository, clocks ...Clock) GetDebtStatement {
	return &getDebtStatementUseCase{repository: repository, clock: clockOrSystem(clocks)}
}

func (useCase *getDebtStatementUseCase) Execute(ctx context.Context, actor Actor, saleID string) (domain.DebtStatement, error) {
	scope, err := collectionScope(actor)
	if err != nil {
		return domain.DebtStatement{}, err
	}
	saleID = strings.TrimSpace(saleID)
	if saleID == "" {
		return domain.DebtStatement{}, domain.ErrSaleNotFound
	}
	statement, err := useCase.repository.GetDebtStatement(ctx, saleID, scope, useCase.clock.Now().UTC())
	if err != nil {
		return domain.DebtStatement{}, fromRepository(err)
	}
	return statement, nil
}
