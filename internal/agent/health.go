package agent

import (
	"context"
	"sync"
	"time"

	"github.com/neogan74/konsul/internal/healthcheck"
	"github.com/neogan74/konsul/internal/logger"
)

// HealthChecker manages health checks for local services
type HealthChecker struct {
	manager      *healthcheck.Manager
	config       HealthCheckConfig
	serverClient *ServerClient
	log          logger.Logger

	// Track previous status for change detection
	lastStatus   map[string]healthcheck.Status
	lastStatusMu sync.RWMutex

	// Metrics
	checksTotal  uint64
	changesTotal uint64
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(cfg HealthCheckConfig, client *ServerClient, log logger.Logger) *HealthChecker {
	return &HealthChecker{
		manager:      healthcheck.NewManager(log),
		config:       cfg,
		serverClient: client,
		log:          log,
		lastStatus:   make(map[string]healthcheck.Status),
	}
}

// Run starts the health checker
func (h *HealthChecker) Run(ctx context.Context) {
	if !h.config.EnableLocalExecution {
		h.log.Info("Local health check execution disabled")
		return
	}

	h.log.Info("Starting health checker",
		logger.String("interval", h.config.CheckInterval.String()))

	// If reporting only changes, monitor for status changes
	if h.config.ReportOnlyChanges {
		go h.monitorStatusChanges(ctx)
	}

	<-ctx.Done()
	h.manager.Stop()
	h.log.Info("Health checker stopped")
}

// RegisterCheck registers a health check for a service
func (h *HealthChecker) RegisterCheck(serviceID string, def *healthcheck.CheckDefinition) error {
	def.ServiceID = serviceID

	// Set default interval and timeout if not specified
	if def.Interval == "" {
		def.Interval = h.config.CheckInterval.String()
	}
	if def.Timeout == "" {
		def.Timeout = h.config.Timeout.String()
	}

	check, err := h.manager.AddCheck(def)
	if err != nil {
		return err
	}

	// Initialize last status
	h.lastStatusMu.Lock()
	h.lastStatus[check.ID] = check.Status
	h.lastStatusMu.Unlock()

	h.log.Info("Health check registered",
		logger.String("check_id", check.ID),
		logger.String("service_id", serviceID),
		logger.String("type", string(check.Type)))

	return nil
}

// DeregisterCheck removes a health check
func (h *HealthChecker) DeregisterCheck(checkID string) error {
	err := h.manager.RemoveCheck(checkID)
	if err != nil {
		return err
	}

	// Remove from last status tracking
	h.lastStatusMu.Lock()
	delete(h.lastStatus, checkID)
	h.lastStatusMu.Unlock()

	h.log.Info("Health check deregistered", logger.String("check_id", checkID))
	return nil
}

// UpdateTTLCheck updates a TTL-based health check
func (h *HealthChecker) UpdateTTLCheck(checkID string) error {
	return h.manager.UpdateTTLCheck(checkID)
}

// GetCheck retrieves a health check by ID
func (h *HealthChecker) GetCheck(checkID string) (*healthcheck.Check, bool) {
	return h.manager.GetCheck(checkID)
}

// ListChecks returns all registered health checks
func (h *HealthChecker) ListChecks() []*healthcheck.Check {
	return h.manager.ListChecks()
}

// GetChecksByService returns all checks for a specific service
func (h *HealthChecker) GetChecksByService(serviceID string) []*healthcheck.Check {
	allChecks := h.manager.ListChecks()
	serviceChecks := make([]*healthcheck.Check, 0)

	for _, check := range allChecks {
		if check.ServiceID == serviceID {
			serviceChecks = append(serviceChecks, check)
		}
	}

	return serviceChecks
}

// monitorStatusChanges monitors for health check status changes and reports them
func (h *HealthChecker) monitorStatusChanges(ctx context.Context) {
	ticker := time.NewTicker(h.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.detectAndReportChanges(ctx)
		}
	}
}

// detectAndReportChanges detects status changes and reports them to the server
func (h *HealthChecker) detectAndReportChanges(ctx context.Context) {
	checks := h.manager.ListChecks()

	h.lastStatusMu.Lock()
	defer h.lastStatusMu.Unlock()

	for _, check := range checks {
		lastStatus, exists := h.lastStatus[check.ID]

		// Check if status changed
		if !exists || lastStatus != check.Status {
			h.log.Info("Health check status changed",
				logger.String("check_id", check.ID),
				logger.String("service_id", check.ServiceID),
				logger.String("old_status", string(lastStatus)),
				logger.String("new_status", string(check.Status)),
				logger.String("output", check.Output))

			// Report change to server
			if h.config.ReportOnlyChanges {
				h.reportStatusChange(ctx, check)
			}

			// Update last status
			h.lastStatus[check.ID] = check.Status
			h.changesTotal++
		}

		h.checksTotal++
	}
}

// reportStatusChange reports a health check status change to the server
func (h *HealthChecker) reportStatusChange(ctx context.Context, check *healthcheck.Check) {
	update := HealthUpdate{
		ServiceID: check.ServiceID,
		CheckID:   check.ID,
		Status:    healthStatusFromCheckStatus(check.Status),
		Output:    check.Output,
		Check: &healthcheck.CheckDefinition{
			ID:        check.ID,
			Name:      check.Name,
			ServiceID: check.ServiceID,
		},
	}

	// Send to server with timeout
	reportCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := h.serverClient.ReportHealthCheck(reportCtx, update); err != nil {
		h.log.Warn("Failed to report health check status change",
			logger.String("check_id", check.ID),
			logger.Error(err))
	} else {
		h.log.Debug("Health check status reported",
			logger.String("check_id", check.ID),
			logger.String("status", string(check.Status)))
	}
}

// GetStats returns health checker statistics
func (h *HealthChecker) GetStats() HealthCheckerStats {
	h.lastStatusMu.RLock()
	defer h.lastStatusMu.RUnlock()

	checks := h.manager.ListChecks()

	passingCount := 0
	warningCount := 0
	criticalCount := 0

	for _, check := range checks {
		switch check.Status {
		case healthcheck.StatusPassing:
			passingCount++
		case healthcheck.StatusWarning:
			warningCount++
		case healthcheck.StatusCritical:
			criticalCount++
		}
	}

	return HealthCheckerStats{
		TotalChecks:    len(checks),
		PassingChecks:  passingCount,
		WarningChecks:  warningCount,
		CriticalChecks: criticalCount,
		ChecksTotal:    h.checksTotal,
		ChangesTotal:   h.changesTotal,
	}
}

// HealthCheckerStats represents health checker statistics
type HealthCheckerStats struct {
	TotalChecks    int    `json:"total_checks"`
	PassingChecks  int    `json:"passing_checks"`
	WarningChecks  int    `json:"warning_checks"`
	CriticalChecks int    `json:"critical_checks"`
	ChecksTotal    uint64 `json:"checks_total"`
	ChangesTotal   uint64 `json:"changes_total"`
}

// healthStatusFromCheckStatus converts healthcheck.Status to agent HealthStatus
func healthStatusFromCheckStatus(status healthcheck.Status) HealthStatus {
	switch status {
	case healthcheck.StatusPassing:
		return HealthStatusPassing
	case healthcheck.StatusWarning:
		return HealthStatusWarning
	case healthcheck.StatusCritical:
		return HealthStatusCritical
	default:
		return HealthStatusUnknown
	}
}
