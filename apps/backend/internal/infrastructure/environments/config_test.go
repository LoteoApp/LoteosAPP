package environments_test

import (
	"strings"
	"testing"

	"loteosapp/backend/internal/infrastructure/environments"
)

func TestLoadServer(t *testing.T) {
	t.Run("reads required settings and applies defaults", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://user:pass@pooler.supabase.com:5432/postgres")
		t.Setenv("SUPABASE_URL", "https://project.supabase.co")
		t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "service-role-key")

		cfg, err := environments.LoadServer()
		if err != nil {
			t.Fatalf("LoadServer() error = %v", err)
		}

		if cfg.DatabaseURL != "postgres://user:pass@pooler.supabase.com:5432/postgres" {
			t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
		}
		if cfg.SupabaseURL != "https://project.supabase.co" {
			t.Errorf("SupabaseURL = %q, want %q", cfg.SupabaseURL, "https://project.supabase.co")
		}
		if cfg.SupabaseServiceRoleKey != "service-role-key" {
			t.Errorf("SupabaseServiceRoleKey = %q, want %q", cfg.SupabaseServiceRoleKey, "service-role-key")
		}
		if cfg.Port != "8080" {
			t.Errorf("Port = %q, want %q", cfg.Port, "8080")
		}
		if cfg.FrontendOrigin != "http://localhost:5173" {
			t.Errorf("FrontendOrigin = %q, want %q", cfg.FrontendOrigin, "http://localhost:5173")
		}
	})

	t.Run("overrides defaults from the environment", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/loteosapp")
		t.Setenv("SUPABASE_URL", "https://project.supabase.co")
		t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "service-role-key")
		t.Setenv("FRONTEND_ORIGIN", "https://app.loteosapp.com")
		t.Setenv("PORT", "9090")

		cfg, err := environments.LoadServer()
		if err != nil {
			t.Fatalf("LoadServer() error = %v", err)
		}

		if cfg.DatabaseURL != "postgres://user:pass@db:5432/loteosapp" {
			t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
		}
		if cfg.FrontendOrigin != "https://app.loteosapp.com" {
			t.Errorf("FrontendOrigin = %q", cfg.FrontendOrigin)
		}
		if cfg.Port != "9090" {
			t.Errorf("Port = %q, want %q", cfg.Port, "9090")
		}
	})

	missing := []struct {
		name         string
		databaseURL  string
		url          string
		serviceRole  string
		wantInErrMsg string
	}{
		{
			name:         "missing database url",
			url:          "https://project.supabase.co",
			serviceRole:  "service-role-key",
			wantInErrMsg: "DATABASE_URL",
		},
		{
			name:         "missing supabase url",
			databaseURL:  "postgres://user:pass@pooler.supabase.com:5432/postgres",
			serviceRole:  "service-role-key",
			wantInErrMsg: "SUPABASE_URL",
		},
		{
			name:         "missing service role key",
			databaseURL:  "postgres://user:pass@pooler.supabase.com:5432/postgres",
			url:          "https://project.supabase.co",
			wantInErrMsg: "SUPABASE_SERVICE_ROLE_KEY",
		},
		{
			name:         "missing all",
			wantInErrMsg: "DATABASE_URL",
		},
	}

	for _, tt := range missing {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", tt.databaseURL)
			t.Setenv("SUPABASE_URL", tt.url)
			t.Setenv("SUPABASE_SERVICE_ROLE_KEY", tt.serviceRole)

			_, err := environments.LoadServer()
			if err == nil {
				t.Fatal("LoadServer() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), tt.wantInErrMsg) {
				t.Errorf("LoadServer() error = %q, want it to mention %q", err, tt.wantInErrMsg)
			}
		})
	}
}

func TestLoadMigration(t *testing.T) {
	t.Run("applies defaults", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://user:pass@pooler.supabase.com:5432/postgres")

		cfg, err := environments.LoadMigration()
		if err != nil {
			t.Fatalf("LoadMigration() error = %v", err)
		}

		if cfg.MigrationsDir != "migrations" {
			t.Errorf("MigrationsDir = %q, want %q", cfg.MigrationsDir, "migrations")
		}
		if cfg.DatabaseURL != "postgres://user:pass@pooler.supabase.com:5432/postgres" {
			t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
		}
	})

	t.Run("overrides defaults from the environment", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/loteosapp")
		t.Setenv("MIGRATIONS_DIR", "/workspace/migrations")

		cfg, err := environments.LoadMigration()
		if err != nil {
			t.Fatalf("LoadMigration() error = %v", err)
		}

		if cfg.DatabaseURL != "postgres://user:pass@db:5432/loteosapp" {
			t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
		}
		if cfg.MigrationsDir != "/workspace/migrations" {
			t.Errorf("MigrationsDir = %q", cfg.MigrationsDir)
		}
	})

	t.Run("missing database url", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")

		_, err := environments.LoadMigration()
		if err == nil {
			t.Fatal("LoadMigration() error = nil, want an error")
		}
		if !strings.Contains(err.Error(), "DATABASE_URL") {
			t.Errorf("LoadMigration() error = %q, want it to mention DATABASE_URL", err)
		}
	})
}

func TestLoadServerDoesNotLeakCredentials(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@pooler.supabase.com:5432/postgres")
	t.Setenv("SUPABASE_URL", "")
	t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "super-secret-service-role-key")

	_, err := environments.LoadServer()
	if err == nil {
		t.Fatal("LoadServer() error = nil, want an error")
	}
	if strings.Contains(err.Error(), "super-secret-service-role-key") {
		t.Errorf("LoadServer() error leaks the service role key: %q", err)
	}
}
