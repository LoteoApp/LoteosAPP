package collections

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type ListDueInstallmentsInput struct {
	Actor  Actor
	Filter domain.DueInstallmentFilter
}

// ListDueInstallments lists the cuotas of active financed ventas by
// vencimiento, across every venta the actor can reach.
type ListDueInstallments interface {
	Execute(ctx context.Context, input ListDueInstallmentsInput) (domain.DueInstallmentPage, error)
}

type listDueInstallmentsUseCase struct {
	repository gateway.CollectionRepository
	users      gateway.UserRepository
	clock      Clock
}

func NewListDueInstallments(repository gateway.CollectionRepository, users gateway.UserRepository, clocks ...Clock) ListDueInstallments {
	return &listDueInstallmentsUseCase{repository: repository, users: users, clock: clockOrSystem(clocks)}
}

func (useCase *listDueInstallmentsUseCase) Execute(ctx context.Context, input ListDueInstallmentsInput) (domain.DueInstallmentPage, error) {
	filter, err := input.Filter.Normalize()
	if err != nil {
		return domain.DueInstallmentPage{}, err
	}
	_, scope, err := authorizeCollector(ctx, useCase.users, input.Actor)
	if err != nil {
		return domain.DueInstallmentPage{}, err
	}
	page, err := useCase.repository.ListDueInstallments(ctx, filter, scope, useCase.clock.Now().UTC())
	if err != nil {
		return domain.DueInstallmentPage{}, fromRepository(err)
	}
	return page, nil
}
