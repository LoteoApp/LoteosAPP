package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/reservations"
)

type ReservationExpiryWorker struct {
	process  reservations.ProcessExpirations
	interval time.Duration
	batch    int
	timeout  time.Duration
}

func NewReservationExpiryWorker(process reservations.ProcessExpirations, interval time.Duration, batch int, timeout time.Duration) *ReservationExpiryWorker {
	return &ReservationExpiryWorker{
		process:  process,
		interval: interval,
		batch:    batch,
		timeout:  timeout,
	}
}

func (worker *ReservationExpiryWorker) Run(ctx context.Context) {
	worker.runOnce(ctx)
	ticker := time.NewTicker(worker.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			worker.runOnce(ctx)
		}
	}
}

func (worker *ReservationExpiryWorker) runOnce(ctx context.Context) {
	operationCtx, cancel := context.WithTimeout(ctx, worker.timeout)
	defer cancel()
	report, err := worker.process.Execute(operationCtx, worker.batch)
	if err != nil {
		logErr := err
		var domainErr *domain.Error
		if errors.As(err, &domainErr) && domainErr.Cause != nil {
			logErr = domainErr.Cause
		}
		slog.ErrorContext(ctx, "reservation expiry run failed", "error", logErr, "code", domainErrCode(err))
		return
	}
	for _, failure := range report.Failures {
		logExpirationFailure(ctx, failure)
	}
	if report.Candidates > 0 {
		slog.InfoContext(ctx, "reservation expiry run completed", "candidates", report.Candidates, "processed", report.Processed, "skipped", report.Skipped, "failures", len(report.Failures))
	}
}

func domainErrCode(err error) string {
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		return domainErr.Code
	}
	return ""
}

func logExpirationFailure(ctx context.Context, failure domain.ExpirationFailure) {
	slog.ErrorContext(ctx, "reservation expiration failed", "reservation_id", failure.ReservationID, "error", failure.Cause)
}
