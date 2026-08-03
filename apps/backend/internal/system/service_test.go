package system

import (
	"context"
	"errors"
	"testing"
)

type repositoryStub struct {
	database    DatabaseInfo
	pool        PoolInfo
	snapshotErr error
	pingErr     error
}

func (repository repositoryStub) Snapshot(context.Context) (DatabaseInfo, PoolInfo, error) {
	return repository.database, repository.pool, repository.snapshotErr
}

func (repository repositoryStub) Ping(context.Context) error {
	return repository.pingErr
}

func TestServiceInfoBuildsConnectedSnapshot(t *testing.T) {
	t.Parallel()

	service := NewService(repositoryStub{
		database: DatabaseInfo{DatabaseName: "loteosapp"},
		pool:     PoolInfo{MaxConnections: 10},
	})

	info, err := service.Info(context.Background())
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	if !info.Database.Connected {
		t.Error("Info() database should be connected")
	}
	if info.Database.DatabaseName != "loteosapp" {
		t.Errorf("Info() database name = %q", info.Database.DatabaseName)
	}
	if info.Pool.MaxConnections != 10 {
		t.Errorf("Info() max connections = %d", info.Pool.MaxConnections)
	}
	if info.CheckedAt.IsZero() {
		t.Error("Info() checked time should be set")
	}
}

func TestServicePropagatesRepositoryErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("database unavailable")
	service := NewService(repositoryStub{snapshotErr: wantErr, pingErr: wantErr})

	if _, err := service.Info(context.Background()); !errors.Is(err, wantErr) {
		t.Errorf("Info() error = %v, want %v", err, wantErr)
	}
	if err := service.Ready(context.Background()); !errors.Is(err, wantErr) {
		t.Errorf("Ready() error = %v, want %v", err, wantErr)
	}
}
