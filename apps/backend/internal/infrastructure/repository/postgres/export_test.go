package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

func SetAfterStatementInstallments(repository *CollectionRepository, hook func()) {
	repository.afterStatementInstallments = hook
}

var ErrRetrySaleWrite = errRetrySaleWrite

func SetAfterConversionWrite(repository *SaleRepository, hook func(context.Context, pgx.Tx) error) {
	repository.afterConversionWrite = hook
}

func ReconcileConversion(ctx context.Context, repository *SaleRepository, command gateway.ConvertReservationCommand, original error) (domain.Sale, error) {
	return repository.reconcileConversion(ctx, command, original)
}
