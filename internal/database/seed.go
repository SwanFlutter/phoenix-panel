package database

import (
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/phoenix-panel/phoenix/internal/config"
	"github.com/phoenix-panel/phoenix/internal/models"
	"github.com/phoenix-panel/phoenix/internal/security"
)

// Seed bootstraps a fresh database: creates the first sudo admin and a local
// node if none exist. It is idempotent and safe to run on every startup.
func Seed(db *gorm.DB, cfg *config.Config) error {
	if err := seedAdmin(db, cfg); err != nil {
		return err
	}
	if err := seedLocalNode(db, cfg); err != nil {
		return err
	}
	return nil
}

func seedAdmin(db *gorm.DB, cfg *config.Config) error {
	var count int64
	if err := db.Model(&models.Admin{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count admins: %w", err)
	}
	if count > 0 {
		return nil // already bootstrapped
	}

	if cfg.Admin.Password == "" {
		return errors.New("no admins exist and PHOENIX_ADMIN_PASSWORD is empty; set it to bootstrap the first admin")
	}

	hash, err := security.HashPassword(cfg.Admin.Password)
	if err != nil {
		return fmt.Errorf("hash bootstrap admin password: %w", err)
	}
	admin := models.Admin{
		Username:     cfg.Admin.Username,
		PasswordHash: hash,
		Role:         models.RoleSudo,
		IsActive:     true,
	}
	if err := db.Create(&admin).Error; err != nil {
		return fmt.Errorf("create bootstrap admin: %w", err)
	}
	slog.Warn("bootstrap sudo admin created — change the password immediately",
		"username", admin.Username)
	return nil
}

func seedLocalNode(db *gorm.DB, cfg *config.Config) error {
	var count int64
	if err := db.Model(&models.Node{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count nodes: %w", err)
	}
	if count > 0 {
		return nil
	}
	node := models.Node{
		Name:     "local",
		Address:  "127.0.0.1",
		APIHost:  "127.0.0.1",
		Core:     models.CoreType(cfg.Core.DefaultCore),
		Status:   models.NodeUnknown,
		IsLocal:  true,
		IsActive: true,
	}
	if err := db.Create(&node).Error; err != nil {
		return fmt.Errorf("create local node: %w", err)
	}
	slog.Info("local node created", "name", node.Name, "core", node.Core)
	return nil
}
