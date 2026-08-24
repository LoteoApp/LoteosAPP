package system

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestCheckDatabaseReadinessPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("database unavailable")
	checkDatabaseReadiness := NewCheckDatabaseReadiness(&gatewayfake.Repository{PingErr: wantErr})

	if err := checkDatabaseReadiness.Execute(context.Background()); !errors.Is(err, wantErr) {
		t.Errorf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestCheckDatabaseReadinessSucceedsWhenRepositoryIsReachable(t *testing.T) {
	t.Parallel()

	checkDatabaseReadiness := NewCheckDatabaseReadiness(&gatewayfake.Repository{})

	if err := checkDatabaseReadiness.Execute(context.Background()); err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
}
