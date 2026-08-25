package environments

import (
	"fmt"
	"os"
	"strings"
)

type Server struct {
	DatabaseURL            string
	FrontendOrigin         string
	Port                   string
	SupabaseURL            string
	SupabaseServiceRoleKey string
	Storage                Storage
}

// Storage addresses the Cloudflare R2 bucket that holds uploaded files.
type Storage struct {
	Endpoint        string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

type Migration struct {
	DatabaseURL   string
	MigrationsDir string
}

// LoadServer reads the server configuration from the environment. DATABASE_URL,
// the Supabase settings and the Cloudflare R2 credentials have no fallback:
// DATABASE_URL points at a shared Supabase project (see docs/database.md), the
// service_role key grants full administrative access to the auth project, and
// the R2 keys grant read/write access to the file bucket, so all of them must
// always come from the environment and never from a value committed to the
// repository.
func LoadServer() (Server, error) {
	var env environment

	cfg := Server{
		DatabaseURL:            env.required("DATABASE_URL"),
		FrontendOrigin:         envOrDefault("FRONTEND_ORIGIN", "http://localhost:5173"),
		Port:                   envOrDefault("PORT", "8080"),
		SupabaseURL:            env.required("SUPABASE_URL"),
		SupabaseServiceRoleKey: env.required("SUPABASE_SERVICE_ROLE_KEY"),
		Storage: Storage{
			Endpoint:        env.required("CLOUDFLARE_R2_ENDPOINT"),
			Bucket:          env.required("CLOUDFLARE_R2_BUCKET_NAME"),
			AccessKeyID:     env.required("CLOUDFLARE_R2_ACCESS_KEY_ID"),
			SecretAccessKey: env.required("CLOUDFLARE_R2_SECRET_ACCESS_KEY"),
		},
	}

	if err := env.err(); err != nil {
		return Server{}, err
	}

	return cfg, nil
}

func LoadMigration() (Migration, error) {
	var env environment

	cfg := Migration{
		DatabaseURL:   env.required("DATABASE_URL"),
		MigrationsDir: envOrDefault("MIGRATIONS_DIR", "migrations"),
	}

	if err := env.err(); err != nil {
		return Migration{}, err
	}

	return cfg, nil
}

// environment collects every variable that was missing, so one run reports
// all of them instead of failing on the first and hiding the rest. It only
// ever records names, never values, so the error is safe to log.
type environment struct {
	missing []string
}

func (env *environment) required(name string) string {
	value := os.Getenv(name)
	if value == "" {
		env.missing = append(env.missing, name)
	}

	return value
}

func (env *environment) err() error {
	if len(env.missing) == 0 {
		return nil
	}

	return fmt.Errorf("%s must be set", strings.Join(env.missing, ", "))
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}
