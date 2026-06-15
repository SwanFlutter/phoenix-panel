// Package database manages the GORM connection lifecycle and migrations.
package database

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/phoenix-panel/phoenix/internal/config"
	"github.com/phoenix-panel/phoenix/internal/models"
)

// Open establishes a *gorm.DB connection for the configured driver and applies
// pragmatic connection-pool settings.
func Open(cfg *config.Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger:                                   gormLogger(cfg),
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: false,
	}

	var dialector gorm.Dialector
	switch cfg.DB.Driver {
	case "sqlite":
		if err := ensureSQLiteDir(cfg.DB.SQLitePath); err != nil {
			return nil, err
		}
		// _pragma settings: WAL for concurrent reads, busy_timeout to avoid
		// "database is locked", foreign_keys ON for referential integrity.
		dsn := cfg.DB.SQLitePath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
		dialector = sqlite.Open(dsn)
	case "postgres":
		dialector = postgres.Open(cfg.DB.PostgresDSN())
	default:
		return nil, fmt.Errorf("unsupported db driver: %s", cfg.DB.Driver)
	}

	db, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	// SQLite is a single writer; keep the pool modest. Postgres can go wider.
	if cfg.DB.Driver == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
	} else {
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	return db, nil
}

// Migrate runs GORM AutoMigrate for every model.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		return fmt.Errorf("auto-migrate: %w", err)
	}
	slog.Info("database migration complete")
	return nil
}

func ensureSQLiteDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create sqlite dir %s: %w", dir, err)
	}
	return nil
}

func gormLogger(cfg *config.Config) logger.Interface {
	level := logger.Warn
	if cfg.Server.Mode == "debug" {
		level = logger.Info
	}
	return logger.New(
		slogWriter{},
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  level,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
}

// slogWriter adapts GORM's logger output onto slog.
type slogWriter struct{}

func (slogWriter) Printf(format string, args ...any) {
	slog.Debug(fmt.Sprintf(format, args...))
}
