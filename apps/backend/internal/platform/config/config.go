package config

import "os"

type Server struct {
	DatabaseURL    string
	FrontendOrigin string
	Port           string
}

type Migration struct {
	DatabaseURL   string
	MigrationsDir string
}

func LoadServer() Server {
	return Server{
		DatabaseURL:    envOrDefault("DATABASE_URL", defaultDatabaseURL),
		FrontendOrigin: envOrDefault("FRONTEND_ORIGIN", "http://localhost:5173"),
		Port:           envOrDefault("PORT", "8080"),
	}
}

func LoadMigration() Migration {
	return Migration{
		DatabaseURL:   envOrDefault("DATABASE_URL", defaultDatabaseURL),
		MigrationsDir: envOrDefault("MIGRATIONS_DIR", "migrations"),
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}

const defaultDatabaseURL = "postgres://loteosapp:loteosapp@localhost:5432/loteosapp?sslmode=disable"
