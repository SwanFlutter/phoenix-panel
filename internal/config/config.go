// Package config loads PHOENIX PANEL configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the panel.
type Config struct {
	Server   ServerConfig
	DB       DBConfig
	Security SecurityConfig
	Admin    AdminBootstrap
	Rate     RateConfig
	CORS     CORSConfig
	Core     CoreConfig
}

type ServerConfig struct {
	Host    string
	Port    int
	BaseURL string
	Mode    string // debug | release
}

type DBConfig struct {
	Driver     string // sqlite | postgres
	SQLitePath string
	Host       string
	Port       int
	User       string
	Password   string
	Name       string
	SSLMode    string
}

type SecurityConfig struct {
	JWTSecret  string
	JWTTTL     time.Duration
	BcryptCost int
}

type AdminBootstrap struct {
	Username string
	Password string
}

type RateConfig struct {
	RPS        float64
	Burst      int
	LoginRPS   float64
	LoginBurst int
}

type CORSConfig struct {
	Origins []string
}

type CoreConfig struct {
	DefaultCore string // xray | sing-box
}

// Load reads configuration from a .env file (if present) and the environment.
// Environment variables always take precedence over the .env file.
func Load() (*Config, error) {
	// Best-effort: .env is optional (e.g. in Docker the env is injected directly).
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Host:    getEnv("PHOENIX_HOST", "0.0.0.0"),
			Port:    getEnvInt("PHOENIX_PORT", 8080),
			BaseURL: strings.TrimRight(getEnv("PHOENIX_BASE_URL", "http://localhost:8080"), "/"),
			Mode:    getEnv("PHOENIX_MODE", "release"),
		},
		DB: DBConfig{
			Driver:     getEnv("PHOENIX_DB_DRIVER", "sqlite"),
			SQLitePath: getEnv("PHOENIX_DB_SQLITE_PATH", "./data/phoenix.db"),
			Host:       getEnv("PHOENIX_DB_HOST", "localhost"),
			Port:       getEnvInt("PHOENIX_DB_PORT", 5432),
			User:       getEnv("PHOENIX_DB_USER", "phoenix"),
			Password:   getEnv("PHOENIX_DB_PASSWORD", ""),
			Name:       getEnv("PHOENIX_DB_NAME", "phoenix"),
			SSLMode:    getEnv("PHOENIX_DB_SSLMODE", "disable"),
		},
		Security: SecurityConfig{
			JWTSecret:  getEnv("PHOENIX_JWT_SECRET", ""),
			JWTTTL:     getEnvDuration("PHOENIX_JWT_TTL", 24*time.Hour),
			BcryptCost: getEnvInt("PHOENIX_BCRYPT_COST", 12),
		},
		Admin: AdminBootstrap{
			Username: getEnv("PHOENIX_ADMIN_USERNAME", "admin"),
			Password: getEnv("PHOENIX_ADMIN_PASSWORD", ""),
		},
		Rate: RateConfig{
			RPS:        getEnvFloat("PHOENIX_RATE_RPS", 20),
			Burst:      getEnvInt("PHOENIX_RATE_BURST", 40),
			LoginRPS:   getEnvFloat("PHOENIX_LOGIN_RATE_RPS", 1),
			LoginBurst: getEnvInt("PHOENIX_LOGIN_RATE_BURST", 5),
		},
		CORS: CORSConfig{
			Origins: splitCSV(getEnv("PHOENIX_CORS_ORIGINS", "*")),
		},
		Core: CoreConfig{
			DefaultCore: getEnv("PHOENIX_DEFAULT_CORE", "xray"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.Security.JWTSecret == "" {
		return fmt.Errorf("PHOENIX_JWT_SECRET is required (generate with: openssl rand -hex 32)")
	}
	if len(c.Security.JWTSecret) < 32 {
		return fmt.Errorf("PHOENIX_JWT_SECRET must be at least 32 characters")
	}
	switch c.DB.Driver {
	case "sqlite", "postgres":
	default:
		return fmt.Errorf("PHOENIX_DB_DRIVER must be 'sqlite' or 'postgres', got %q", c.DB.Driver)
	}
	switch c.Core.DefaultCore {
	case "xray", "sing-box":
	default:
		return fmt.Errorf("PHOENIX_DEFAULT_CORE must be 'xray' or 'sing-box', got %q", c.Core.DefaultCore)
	}
	if c.Security.BcryptCost < 10 || c.Security.BcryptCost > 15 {
		return fmt.Errorf("PHOENIX_BCRYPT_COST must be between 10 and 15")
	}
	return nil
}

// PostgresDSN builds a libpq-style connection string.
func (d DBConfig) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// ---- env helpers ----

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
