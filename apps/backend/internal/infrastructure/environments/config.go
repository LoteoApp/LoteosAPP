package environments

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	DatabaseURL            string
	FrontendOrigin         string
	Port                   string
	SupabaseURL            string
	SupabaseServiceRoleKey string
	Storage                Storage
	ReservationExpiry      ReservationExpiry
}

type ReservationExpiry struct {
	Enabled  bool
	Interval time.Duration
	Batch    int
	Timeout  time.Duration
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
	worker, err := loadReservationExpiry(&env)

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
		ReservationExpiry: worker,
	}

	if err != nil {
		return Server{}, err
	}
	if err := env.err(); err != nil {
		return Server{}, err
	}

	return cfg, nil
}

func loadReservationExpiry(env *environment) (ReservationExpiry, error) {
	enabled, err := strconv.ParseBool(envOrDefault("RESERVATION_EXPIRY_ENABLED", "true"))
	if err != nil {
		return ReservationExpiry{}, fmt.Errorf("RESERVATION_EXPIRY_ENABLED must be true or false")
	}
	interval, err := time.ParseDuration(envOrDefault("RESERVATION_EXPIRY_INTERVAL", "1m"))
	if err != nil || interval <= 0 {
		return ReservationExpiry{}, fmt.Errorf("RESERVATION_EXPIRY_INTERVAL must be a positive duration")
	}
	batch, err := strconv.Atoi(envOrDefault("RESERVATION_EXPIRY_BATCH", "50"))
	if err != nil || batch < 1 {
		return ReservationExpiry{}, fmt.Errorf("RESERVATION_EXPIRY_BATCH must be a positive integer")
	}
	timeout, err := time.ParseDuration(envOrDefault("RESERVATION_EXPIRY_TIMEOUT", "10s"))
	if err != nil || timeout <= 0 {
		return ReservationExpiry{}, fmt.Errorf("RESERVATION_EXPIRY_TIMEOUT must be a positive duration")
	}
	return ReservationExpiry{Enabled: enabled, Interval: interval, Batch: batch, Timeout: timeout}, nil
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
