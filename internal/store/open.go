package store

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/internal/store/filestore"
	"github.com/monoposer/lowcode-bpmn/internal/store/gormstore"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

// Backend identifies a persistence implementation.
type Backend string

const (
	BackendDB     Backend = "db"
	BackendFile   Backend = "file"
	BackendMemory Backend = "memory"
)

// Config selects which store backend to open.
type Config struct {
	Backend     Backend
	FilePath    string
	DBDriver    gormstore.Driver
	DatabaseURL string
}

// LoadConfig reads store settings from environment variables.
func LoadConfig() (Config, error) {
	backend, err := ParseBackend(getEnv("STORE_BACKEND", "db"))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Backend:     backend,
		FilePath:    getEnv("STORE_PATH", "./data"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	driver, err := gormstore.ParseDriver(getEnv("DB_DRIVER", "postgres"))
	if err != nil {
		return Config{}, err
	}
	cfg.DBDriver = driver
	return cfg, nil
}

// ParseBackend normalizes a backend name.
func ParseBackend(raw string) (Backend, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "db", "database", "sql", "gorm":
		return BackendDB, nil
	case "file", "yaml", "yml":
		return BackendFile, nil
	case "memory", "mem":
		return BackendMemory, nil
	default:
		return "", fmt.Errorf("unsupported STORE_BACKEND %q (use db, file, or memory)", raw)
	}
}

// Open constructs the configured engine store.
func Open(ctx context.Context, cfg Config) (engine.Store, error) {
	switch cfg.Backend {
	case BackendFile:
		return filestore.Open(cfg.FilePath)
	case BackendMemory:
		return memstore.NewStore(), nil
	case BackendDB:
		if cfg.DatabaseURL == "" {
			return nil, fmt.Errorf("DATABASE_URL is required when STORE_BACKEND=db")
		}
		db, err := gormstore.Open(cfg.DBDriver, cfg.DatabaseURL)
		if err != nil {
			return nil, err
		}
		return gormstore.NewStore(ctx, db)
	default:
		return nil, fmt.Errorf("unsupported store backend %q", cfg.Backend)
	}
}

// Ping verifies store connectivity when supported.
func Ping(ctx context.Context, st engine.Store) error {
	switch s := st.(type) {
	case interface{ Ping(context.Context) error }:
		return s.Ping(ctx)
	default:
		return nil
	}
}

// Describe returns a short label for logs.
func Describe(cfg Config) string {
	switch cfg.Backend {
	case BackendFile:
		return fmt.Sprintf("file(%s)", cfg.FilePath)
	case BackendMemory:
		return "memory"
	default:
		return fmt.Sprintf("db(%s)", cfg.DBDriver)
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
