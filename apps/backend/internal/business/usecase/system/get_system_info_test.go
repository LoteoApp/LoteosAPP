package system

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestGetSystemInfoBuildsConnectedSnapshot(t *testing.T) {
	t.Parallel()

	getSystemInfo := NewGetSystemInfo(&gatewayfake.Repository{
		Database: domain.DatabaseInfo{DatabaseName: "loteosapp"},
		Pool:     domain.PoolInfo{MaxConnections: 10},
	})

	got, err := getSystemInfo.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !got.Database.Connected {
		t.Error("Execute() database should be connected")
	}
	if got.Database.DatabaseName != "loteosapp" {
		t.Errorf("Execute() database name = %q", got.Database.DatabaseName)
	}
	if got.Pool.MaxConnections != 10 {
		t.Errorf("Execute() max connections = %d", got.Pool.MaxConnections)
	}
	if got.CheckedAt.IsZero() {
		t.Error("Execute() checked time should be set")
	}
}

func TestGetSystemInfoPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("database unavailable")
	getSystemInfo := NewGetSystemInfo(&gatewayfake.Repository{SnapshotErr: wantErr})

	if _, err := getSystemInfo.Execute(context.Background()); !errors.Is(err, wantErr) {
		t.Errorf("Execute() error = %v, want %v", err, wantErr)
	}
}
