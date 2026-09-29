package gatewayfake

import (
	"context"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// CollectionRepository is a fake gateway.CollectionRepository for tests.
type CollectionRepository struct {
	GetDebtStatementCalls  int
	GetDebtStatementErr    error
	GetDebtStatementResult domain.DebtStatement
	GetDebtStatementID     string
	GetDebtStatementScope  gateway.SaleScope
	GetDebtStatementNow    time.Time

	ListDueInstallmentsCalls  int
	ListDueInstallmentsErr    error
	ListDueInstallmentsResult domain.DueInstallmentPage
	ListDueInstallmentsFilter domain.DueInstallmentFilter
	ListDueInstallmentsScope  gateway.SaleScope
	ListDueInstallmentsNow    time.Time

	RegisterPaymentCalls   int
	RegisterPaymentErr     error
	RegisterPaymentResult  domain.Payment
	RegisterPaymentCommand gateway.RegisterPaymentCommand
	RegisterPaymentScope   gateway.SaleScope
}

func (fake *CollectionRepository) GetDebtStatement(_ context.Context, saleID string, scope gateway.SaleScope, now time.Time) (domain.DebtStatement, error) {
	fake.GetDebtStatementCalls++
	fake.GetDebtStatementID = saleID
	fake.GetDebtStatementScope = scope
	fake.GetDebtStatementNow = now
	if fake.GetDebtStatementErr != nil {
		return domain.DebtStatement{}, fake.GetDebtStatementErr
	}
	return fake.GetDebtStatementResult, nil
}

func (fake *CollectionRepository) ListDueInstallments(_ context.Context, filter domain.DueInstallmentFilter, scope gateway.SaleScope, now time.Time) (domain.DueInstallmentPage, error) {
	fake.ListDueInstallmentsCalls++
	fake.ListDueInstallmentsFilter = filter
	fake.ListDueInstallmentsScope = scope
	fake.ListDueInstallmentsNow = now
	if fake.ListDueInstallmentsErr != nil {
		return domain.DueInstallmentPage{}, fake.ListDueInstallmentsErr
	}
	return fake.ListDueInstallmentsResult, nil
}

func (fake *CollectionRepository) RegisterPayment(_ context.Context, command gateway.RegisterPaymentCommand, scope gateway.SaleScope) (domain.Payment, error) {
	fake.RegisterPaymentCalls++
	fake.RegisterPaymentCommand = command
	fake.RegisterPaymentScope = scope
	if fake.RegisterPaymentErr != nil {
		return domain.Payment{}, fake.RegisterPaymentErr
	}
	return fake.RegisterPaymentResult, nil
}
