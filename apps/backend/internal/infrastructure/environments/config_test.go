package environments_test

import (
	"strings"
	"testing"

	"loteosapp/backend/internal/infrastructure/environments"
)

const (
	testDatabaseURL    = "postgres://user:pass@pooler.supabase.com:5432/postgres"
	testSupabaseURL    = "https://project.supabase.co"
	testServiceRoleKey = "service-role-key"
	testR2Endpoint     = "https://account.r2.cloudflarestorage.com"
	testR2Bucket       = "loteos-files-dev"
	testR2AccessKey    = "r2-access-key-id"
	testR2SecretAccess = "r2-secret-access-key"
)

// setRequiredServerEnv sets every variable LoadServer requires, so a test
// that exercises one of them starts from an otherwise valid environment.
func setRequiredServerEnv(t *testing.T) {
	t.Helper()

	t.Setenv("DATABASE_URL", testDatabaseURL)
	t.Setenv("SUPABASE_URL", testSupabaseURL)
	t.Setenv("SUPABASE_SERVICE_ROLE_KEY", testServiceRoleKey)
	t.Setenv("CLOUDFLARE_R2_ENDPOINT", testR2Endpoint)
	t.Setenv("CLOUDFLARE_R2_BUCKET_NAME", testR2Bucket)
	t.Setenv("CLOUDFLARE_R2_ACCESS_KEY_ID", testR2AccessKey)
	t.Setenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY", testR2SecretAccess)
}

func TestLoadServer(t *testing.T) {
	t.Run("reads required settings and applies defaults", func(t *testing.T) {
		setRequiredServerEnv(t)

		cfg, err := environments.LoadServer()
		if err != nil {
			t.Fatalf("LoadServer() error = %v", err)
		}

		if cfg.DatabaseURL != testDatabaseURL {
			t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
		}
		if cfg.SupabaseURL != testSupabaseURL {
			t.Errorf("SupabaseURL = %q, want %q", cfg.SupabaseURL, testSupabaseURL)
		}
		if cfg.SupabaseServiceRoleKey != testServiceRoleKey {
			t.Errorf("SupabaseServiceRoleKey = %q, want %q", cfg.SupabaseServiceRoleKey, testServiceRoleKey)
		}
		if cfg.Port != "8080" {
			t.Errorf("Port = %q, want %q", cfg.Port, "8080")
		}
		if cfg.FrontendOrigin != "http://localhost:5173" {
			t.Errorf("FrontendOrigin = %q, want %q", cfg.FrontendOrigin, "http://localhost:5173")
		}
	})

	t.Run("reads the storage settings", func(t *testing.T) {
		setRequiredServerEnv(t)

		cfg, err := environments.LoadServer()
		if err != nil {
			t.Fatalf("LoadServer() error = %v", err)
		}

		want := environments.Storage{
			Endpoint:        testR2Endpoint,
			Bucket:          testR2Bucket,
			AccessKeyID:     testR2AccessKey,
			SecretAccessKey: testR2SecretAccess,
		}
		if cfg.Storage != want {
			t.Errorf("Storage = %+v, want %+v", cfg.Storage, want)
		}
	})

	t.Run("overrides defaults from the environment", func(t *testing.T) {
		setRequiredServerEnv(t)
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

	required := []string{
		"DATABASE_URL",
		"SUPABASE_URL",
		"SUPABASE_SERVICE_ROLE_KEY",
		"CLOUDFLARE_R2_ENDPOINT",
		"CLOUDFLARE_R2_BUCKET_NAME",
		"CLOUDFLARE_R2_ACCESS_KEY_ID",
		"CLOUDFLARE_R2_SECRET_ACCESS_KEY",
	}

	for _, name := range required {
		t.Run("missing "+name, func(t *testing.T) {
			setRequiredServerEnv(t)
			t.Setenv(name, "")

			_, err := environments.LoadServer()
			if err == nil {
				t.Fatal("LoadServer() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), name) {
				t.Errorf("LoadServer() error = %q, want it to mention %q", err, name)
			}
		})
	}

	t.Run("reports every missing variable at once", func(t *testing.T) {
		for _, name := range required {
			t.Setenv(name, "")
		}

		_, err := environments.LoadServer()
		if err == nil {
			t.Fatal("LoadServer() error = nil, want an error")
		}
		for _, name := range required {
			if !strings.Contains(err.Error(), name) {
				t.Errorf("LoadServer() error = %q, want it to mention %q", err, name)
			}
		}
	})
}

func TestLoadMigration(t *testing.T) {
	t.Run("applies defaults", func(t *testing.T) {
		t.Setenv("DATABASE_URL", testDatabaseURL)

		cfg, err := environments.LoadMigration()
		if err != nil {
			t.Fatalf("LoadMigration() error = %v", err)
		}

		if cfg.MigrationsDir != "migrations" {
			t.Errorf("MigrationsDir = %q, want %q", cfg.MigrationsDir, "migrations")
		}
		if cfg.DatabaseURL != testDatabaseURL {
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
	secrets := map[string]string{
		"SUPABASE_SERVICE_ROLE_KEY":       "super-secret-service-role-key",
		"CLOUDFLARE_R2_SECRET_ACCESS_KEY": "super-secret-r2-key",
	}

	for name, secret := range secrets {
		t.Run(name, func(t *testing.T) {
			setRequiredServerEnv(t)
			t.Setenv(name, secret)
			// Force the failure on a different variable, so the error is
			// built while the secret is present in the environment.
			t.Setenv("SUPABASE_URL", "")

			_, err := environments.LoadServer()
			if err == nil {
				t.Fatal("LoadServer() error = nil, want an error")
			}
			if strings.Contains(err.Error(), secret) {
				t.Errorf("LoadServer() error leaks %s: %q", name, err)
			}
		})
	}
}
