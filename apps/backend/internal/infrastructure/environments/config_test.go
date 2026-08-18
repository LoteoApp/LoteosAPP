package environments_test

import (
	"strings"
	"testing"

	"loteosapp/backend/internal/infrastructure/environments"
)

func TestLoadServer(t *testing.T) {
	t.Run("reads supabase settings and applies defaults", func(t *testing.T) {
		t.Setenv("SUPABASE_URL", "https://project.supabase.co")
		t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "service-role-key")

		cfg, err := environments.LoadServer()
		if err != nil {
			t.Fatalf("LoadServer() error = %v", err)
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
		if cfg.DatabaseURL == "" {
			t.Error("DatabaseURL is empty, want the local development default")
		}
	})

	t.Run("overrides defaults from the environment", func(t *testing.T) {
		t.Setenv("SUPABASE_URL", "https://project.supabase.co")
		t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "service-role-key")
		t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/loteosapp")
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
		url          string
		serviceRole  string
		wantInErrMsg string
	}{
		{
			name:         "missing supabase url",
			serviceRole:  "service-role-key",
			wantInErrMsg: "SUPABASE_URL",
		},
		{
			name:         "missing service role key",
			url:          "https://project.supabase.co",
			wantInErrMsg: "SUPABASE_SERVICE_ROLE_KEY",
		},
		{
			name:         "missing both",
			wantInErrMsg: "SUPABASE_URL",
		},
	}

	for _, tt := range missing {
		t.Run(tt.name, func(t *testing.T) {
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
		cfg := environments.LoadMigration()

		if cfg.MigrationsDir != "migrations" {
			t.Errorf("MigrationsDir = %q, want %q", cfg.MigrationsDir, "migrations")
		}
		if cfg.DatabaseURL == "" {
			t.Error("DatabaseURL is empty, want the local development default")
		}
	})

	t.Run("overrides defaults from the environment", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/loteosapp")
		t.Setenv("MIGRATIONS_DIR", "/workspace/migrations")

		cfg := environments.LoadMigration()

		if cfg.DatabaseURL != "postgres://user:pass@db:5432/loteosapp" {
			t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
		}
		if cfg.MigrationsDir != "/workspace/migrations" {
			t.Errorf("MigrationsDir = %q", cfg.MigrationsDir)
		}
	})
}

func TestLoadServerDoesNotLeakCredentials(t *testing.T) {
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
