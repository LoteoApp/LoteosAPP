package environments

import "os"

type Server struct {
	DatabaseURL          string
	FrontendOrigin       string
	Port                 string
	KeycloakIssuer       string
	KeycloakJWKSBaseURL  string
	KeycloakAudience     string
	KeycloakBaseURL      string
	KeycloakRealm        string
	KeycloakClientSecret string
}

type Migration struct {
	DatabaseURL   string
	MigrationsDir string
}

func LoadServer() Server {
	return Server{
		DatabaseURL:          envOrDefault("DATABASE_URL", defaultDatabaseURL),
		FrontendOrigin:       envOrDefault("FRONTEND_ORIGIN", "http://localhost:5173"),
		Port:                 envOrDefault("PORT", "8080"),
		KeycloakIssuer:       envOrDefault("KEYCLOAK_ISSUER", defaultKeycloakIssuer),
		KeycloakJWKSBaseURL:  envOrDefault("KEYCLOAK_JWKS_BASE_URL", defaultKeycloakIssuer),
		KeycloakAudience:     envOrDefault("KEYCLOAK_AUDIENCE", defaultKeycloakAudience),
		KeycloakBaseURL:      envOrDefault("KEYCLOAK_BASE_URL", defaultKeycloakBaseURL),
		KeycloakRealm:        envOrDefault("KEYCLOAK_REALM", defaultKeycloakRealm),
		KeycloakClientSecret: envOrDefault("KEYCLOAK_CLIENT_SECRET", defaultKeycloakClientSecret),
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

const defaultKeycloakIssuer = "http://localhost:8081/realms/loteosapp"
const defaultKeycloakAudience = "loteosapp-backend"
const defaultKeycloakBaseURL = "http://localhost:8081"
const defaultKeycloakRealm = "loteosapp"
const defaultKeycloakClientSecret = "loteosapp-backend-dev-secret"
