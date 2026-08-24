package environments

import (
	"fmt"
	"os"
)

type Server struct {
	DatabaseURL            string
	FrontendOrigin         string
	Port                   string
	SupabaseURL            string
	SupabaseServiceRoleKey string
}

type Migration struct {
	DatabaseURL   string
	MigrationsDir string
}

// LoadServer reads the server configuration from the environment. DATABASE_URL
// and the Supabase settings have no fallback: DATABASE_URL points at a shared
// Supabase project (see docs/database.md), and the service_role key grants
// full administrative access to the auth project, so both must always come
// from the environment and never from a value committed to the repository.
func LoadServer() (Server, error) {
	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return Server{}, err
	}

	supabaseURL, err := requiredEnv("SUPABASE_URL")
	if err != nil {
		return Server{}, err
	}

	serviceRoleKey, err := requiredEnv("SUPABASE_SERVICE_ROLE_KEY")
	if err != nil {
		return Server{}, err
	}

	return Server{
		DatabaseURL:            databaseURL,
		FrontendOrigin:         envOrDefault("FRONTEND_ORIGIN", "http://localhost:5173"),
		Port:                   envOrDefault("PORT", "8080"),
		SupabaseURL:            supabaseURL,
		SupabaseServiceRoleKey: serviceRoleKey,
	}, nil
}

func LoadMigration() (Migration, error) {
	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return Migration{}, err
	}

	return Migration{
		DatabaseURL:   databaseURL,
		MigrationsDir: envOrDefault("MIGRATIONS_DIR", "migrations"),
	}, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}

func requiredEnv(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("%s must be set", name)
	}

	return value, nil
}
