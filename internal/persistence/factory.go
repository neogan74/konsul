package persistence

import (
	"fmt"

	"github.com/neogan74/konsul/internal/logger"
)

// NewEngine creates a persistence engine based on configuration
func NewEngine(cfg Config, log logger.Logger) (Engine, error) {
	if !cfg.Enabled {
		log.Info("Persistence disabled, using in-memory storage")
		return NewMemoryEngine(), nil
	}

	switch cfg.Type {
	case "memory":
		log.Info("Using in-memory persistence")
		return NewMemoryEngine(), nil
	case "badger":
		log.Info("Using BadgerDB persistence",
			logger.String("data_dir", cfg.DataDir),
			logger.String("sync_writes", fmt.Sprintf("%t", cfg.SyncWrites)))
		return NewBadgerEngine(cfg.DataDir, cfg.SyncWrites, log)
	default:
		return nil, fmt.Errorf("unsupported persistence type: %s", cfg.Type)
	}
}
