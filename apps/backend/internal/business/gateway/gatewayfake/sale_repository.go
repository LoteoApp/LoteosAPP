package gatewayfake

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// SaleRepository is a fake gateway.SaleRepository for tests.
type SaleRepository struct {
	CreateCalls   int
	CreateErr     error
	CreateResult  domain.Sale
	CreateCommand gateway.CreateSaleCommand

	ListCalls  int
	ListErr    error
	ListResult domain.SalePage
	ListFilter domain.SaleListFilter
	ListScope  gateway.SaleScope

	GetCalls  int
	GetErr    error
	GetResult domain.Sale
	GetID     string
	GetScope  gateway.SaleScope
}

func (fake *SaleRepository) Create(_ context.Context, command gateway.CreateSaleCommand) (domain.Sale, error) {
	fake.CreateCalls++
	fake.CreateCommand = command
	if fake.CreateErr != nil {
		return domain.Sale{}, fake.CreateErr
	}
	return fake.CreateResult, nil
}

func (fake *SaleRepository) List(_ context.Context, filter domain.SaleListFilter, scope gateway.SaleScope) (domain.SalePage, error) {
	fake.ListCalls++
	fake.ListFilter = filter
	fake.ListScope = scope
	if fake.ListErr != nil {
		return domain.SalePage{}, fake.ListErr
	}
	return fake.ListResult, nil
}

func (fake *SaleRepository) Get(_ context.Context, id string, scope gateway.SaleScope) (domain.Sale, error) {
	fake.GetCalls++
	fake.GetID = id
	fake.GetScope = scope
	if fake.GetErr != nil {
		return domain.Sale{}, fake.GetErr
	}
	return fake.GetResult, nil
}
