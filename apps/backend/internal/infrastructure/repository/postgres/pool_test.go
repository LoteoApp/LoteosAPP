package postgres

import (
	"context"
	"testing"
	"time"
)

func TestPoolConfig(t *testing.T) {
	t.Run("bounds the pool for the shared supabase pooler", func(t *testing.T) {
		config, err := poolConfig("postgres://user:password@pooler.supabase.com:5432/postgres?sslmode=require")
		if err != nil {
			t.Fatalf("poolConfig() error = %v", err)
		}

		if config.MaxConns != 4 {
			t.Errorf("MaxConns = %d, want 4", config.MaxConns)
		}
		if config.MinConns != 0 {
			t.Errorf("MinConns = %d, want 0", config.MinConns)
		}
		if config.MaxConnLifetime != time.Hour {
			t.Errorf("MaxConnLifetime = %v, want %v", config.MaxConnLifetime, time.Hour)
		}
		if config.MaxConnIdleTime != 5*time.Minute {
			t.Errorf("MaxConnIdleTime = %v, want %v", config.MaxConnIdleTime, 5*time.Minute)
		}
		if config.HealthCheckPeriod != time.Minute {
			t.Errorf("HealthCheckPeriod = %v, want %v", config.HealthCheckPeriod, time.Minute)
		}
	})

	t.Run("bounds every wait against the remote database", func(t *testing.T) {
		config, err := poolConfig("postgres://user:password@pooler.supabase.com:5432/postgres?sslmode=require")
		if err != nil {
			t.Fatalf("poolConfig() error = %v", err)
		}

		if config.ConnConfig.ConnectTimeout != 5*time.Second {
			t.Errorf("ConnectTimeout = %v, want %v", config.ConnConfig.ConnectTimeout, 5*time.Second)
		}
		if config.ConnConfig.ConnectTimeout <= 0 {
			t.Error("ConnectTimeout must be set, an unbounded dial hangs on the OS timeout")
		}
	})

	t.Run("keeps the sslmode from the connection string", func(t *testing.T) {
		config, err := poolConfig("postgres://user:password@pooler.supabase.com:5432/postgres?sslmode=require")
		if err != nil {
			t.Fatalf("poolConfig() error = %v", err)
		}

		if config.ConnConfig.TLSConfig == nil {
			t.Error("TLSConfig is nil, want the connection string's sslmode=require to be honoured")
		}
	})

	t.Run("rejects an invalid connection string", func(t *testing.T) {
		if _, err := poolConfig("not-a-connection-string"); err == nil {
			t.Fatal("poolConfig() error = nil, want an error")
		}
	})
}

func TestOpenPoolRejectsInvalidConnectionString(t *testing.T) {
	pool, err := OpenPool(context.Background(), "not-a-connection-string")
	if err == nil {
		pool.Close()
		t.Fatal("OpenPool() error = nil, want an error")
	}
	if pool != nil {
		t.Error("OpenPool() returned a pool alongside an error")
	}
}
