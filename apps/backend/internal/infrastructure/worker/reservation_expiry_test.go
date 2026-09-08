package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
)

type expiryProcessStub struct {
	calls     atomic.Int32
	limit     atomic.Int32
	deadline  atomic.Bool
	err       error
	report    domain.ExpirationReport
	onExecute func(context.Context)
}

func (stub *expiryProcessStub) Execute(ctx context.Context, limit int) (domain.ExpirationReport, error) {
	stub.calls.Add(1)
	stub.limit.Store(int32(limit))
	_, hasDeadline := ctx.Deadline()
	stub.deadline.Store(hasDeadline)
	if stub.onExecute != nil {
		stub.onExecute(ctx)
	}
	return stub.report, stub.err
}

func TestReservationExpiryWorkerRunsImmediatelyAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	process := &expiryProcessStub{onExecute: func(context.Context) { cancel() }}
	worker := NewReservationExpiryWorker(process, time.Hour, 7, time.Second)

	worker.Run(ctx)

	if process.calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", process.calls.Load())
	}
	if process.limit.Load() != 7 {
		t.Errorf("limit = %d, want 7", process.limit.Load())
	}
	if !process.deadline.Load() {
		t.Error("worker should bound each run with a timeout")
	}
}

func TestReservationExpiryWorkerLogsAndContinuesAfterRunFailure(t *testing.T) {
	process := &expiryProcessStub{
		err: errors.New("database unavailable"),
		report: domain.ExpirationReport{
			Candidates: 1,
			Failures:   []domain.ExpirationFailure{{ReservationID: "reservation-1", Cause: errors.New("serialization failure")}},
		},
	}
	worker := NewReservationExpiryWorker(process, time.Hour, 3, time.Second)

	worker.runOnce(context.Background())

	if process.calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", process.calls.Load())
	}

	process.err = nil
	worker.runOnce(context.Background())
	if process.calls.Load() != 2 {
		t.Fatalf("calls after successful run = %d, want 2", process.calls.Load())
	}
}
