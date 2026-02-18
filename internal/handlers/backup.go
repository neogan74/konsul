package handlers

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/middleware"
	"github.com/neogan74/konsul/internal/persistence"
)

// BackupHandler handles backup and restore operations
type BackupHandler struct {
	engine persistence.Engine
	log    logger.Logger
}

// NewBackupHandler creates a new backup handler
func NewBackupHandler(engine persistence.Engine, log logger.Logger) *BackupHandler {
	return &BackupHandler{
		engine: engine,
		log:    log,
	}
}

// CreateBackup creates a backup of the current data
func (h *BackupHandler) CreateBackup(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	if h.engine == nil {
		return middleware.BadRequest(c, "Backup not available - persistence is disabled")
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join("./backups", fmt.Sprintf("konsul-backup-%s.db", timestamp))

	if err := h.engine.Backup(backupPath); err != nil {
		log.Error("Failed to create backup", logger.Error(err))
		return middleware.InternalServerError(c, "Failed to create backup")
	}

	log.Info("Backup created successfully", logger.String("path", backupPath))

	return c.JSON(fiber.Map{
		"message":     "Backup created successfully",
		"backup_path": backupPath,
		"timestamp":   timestamp,
	})
}

// RestoreBackup restores data from a backup file
func (h *BackupHandler) RestoreBackup(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	if h.engine == nil {
		return middleware.BadRequest(c, "Restore not available - persistence is disabled")
	}

	var body struct {
		BackupPath string `json:"backup_path"`
	}

	if err := c.BodyParser(&body); err != nil {
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	if body.BackupPath == "" {
		return middleware.BadRequest(c, "backup_path is required")
	}

	if err := h.engine.Restore(body.BackupPath); err != nil {
		log.Error("Failed to restore backup",
			logger.String("backup_path", body.BackupPath),
			logger.Error(err))
		return middleware.InternalServerError(c, "Failed to restore backup")
	}

	log.Info("Backup restored successfully", logger.String("path", body.BackupPath))

	return c.JSON(fiber.Map{
		"message": "Backup restored successfully",
		"path":    body.BackupPath,
	})
}

// ExportData exports all data as JSON
func (h *BackupHandler) ExportData(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	if h.engine == nil {
		return middleware.BadRequest(c, "Export not available - persistence is disabled")
	}

	// Check if engine has export capability
	if badgerEngine, ok := h.engine.(*persistence.BadgerEngine); ok {
		data, err := badgerEngine.ExportData()
		if err != nil {
			log.Error("Failed to export data", logger.Error(err))
			return middleware.InternalServerError(c, "Failed to export data")
		}

		log.Info("Data exported successfully")
		return c.JSON(data)
	}

	return middleware.BadRequest(c, "Export not supported for this persistence type")
}

// ImportData imports data from JSON
func (h *BackupHandler) ImportData(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	if h.engine == nil {
		return middleware.BadRequest(c, "Import not available - persistence is disabled")
	}

	var data map[string]interface{}
	if err := c.BodyParser(&data); err != nil {
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	// Check if engine has import capability
	if badgerEngine, ok := h.engine.(*persistence.BadgerEngine); ok {
		if err := badgerEngine.ImportData(data); err != nil {
			log.Error("Failed to import data", logger.Error(err))
			return middleware.InternalServerError(c, "Failed to import data")
		}

		log.Info("Data imported successfully")
		return c.JSON(fiber.Map{"message": "Data imported successfully"})
	}

	return middleware.BadRequest(c, "Import not supported for this persistence type")
}

// ListBackups lists available backup files
func (h *BackupHandler) ListBackups(c *fiber.Ctx) error {
	if h.engine == nil {
		return middleware.BadRequest(c, "Backup listing not available - persistence is disabled")
	}

	// This is a simplified implementation
	// In a real implementation, you'd scan the backup directory
	return c.JSON(fiber.Map{
		"message": "Backup listing not implemented yet",
		"note":    "Check the ./backups directory for available backup files",
	})
}
