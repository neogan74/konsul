package healthcheck

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/neogan74/konsul/internal/logger"
)

type Manager struct {
	checks map[string]*Check
	mutex  sync.RWMutex
	log    logger.Logger
	ctx    context.Context
	cancel context.CancelFunc
	stopCh chan struct{}

	httpChecker *HTTPChecker
	tcpChecker  *TCPChecker
	grpcChecker *GRPCChecker
}

func NewManager(log logger.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		checks:      make(map[string]*Check),
		log:         log,
		ctx:         ctx,
		cancel:      cancel,
		stopCh:      make(chan struct{}),
		httpChecker: NewHTTPChecker(),
		tcpChecker:  NewTCPChecker(),
		grpcChecker: NewGRPCChecker(),
	}
}

func (m *Manager) AddCheck(def *CheckDefinition) (*Check, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Generate ID if not provided
	if def.ID == "" {
		def.ID = uuid.New().String()
	}

	// Determine check type
	checkType := CheckTypeTTL
	if def.HTTP != "" {
		checkType = CheckTypeHTTP
	} else if def.TCP != "" {
		checkType = CheckTypeTCP
	} else if def.GRPC != "" {
		checkType = CheckTypeGRPC
	} else if def.TTL != "" {
		checkType = CheckTypeTTL
	}

	// Parse durations
	interval := 30 * time.Second
	if def.Interval != "" {
		if parsed, err := time.ParseDuration(def.Interval); err == nil {
			interval = parsed
		}
	}

	timeout := 10 * time.Second
	if def.Timeout != "" {
		if parsed, err := time.ParseDuration(def.Timeout); err == nil {
			timeout = parsed
		}
	}

	ttl := 0 * time.Second
	if def.TTL != "" {
		if parsed, err := time.ParseDuration(def.TTL); err == nil {
			ttl = parsed
		}
	}

	check := &Check{
		ID:            def.ID,
		Name:          def.Name,
		ServiceID:     def.ServiceID,
		Type:          checkType,
		Status:        StatusCritical,
		Interval:      interval,
		Timeout:       timeout,
		HTTP:          def.HTTP,
		Method:        def.Method,
		Headers:       def.Headers,
		TLSSkipVerify: def.TLSSkipVerify,
		TCP:           def.TCP,
		GRPC:          def.GRPC,
		GRPCUseTLS:    def.GRPCUseTLS,
		TTL:           ttl,
		LastCheck:     time.Now(),
	}

	if checkType == CheckTypeTTL && ttl > 0 {
		check.ExpiresAt = time.Now().Add(ttl)
	}

	m.checks[check.ID] = check

	// Start monitoring for non-TTL checks
	if checkType != CheckTypeTTL {
		go m.runCheck(check)
	}

	m.log.Info("Health check added",
		logger.String("id", check.ID),
		logger.String("name", check.Name),
		logger.String("type", string(checkType)),
		logger.String("service", check.ServiceID))

	return check, nil
}

func (m *Manager) GetCheck(id string) (*Check, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	check, exists := m.checks[id]
	if !exists {
		return nil, false
	}

	// For TTL checks, verify if they're still valid
	if check.Type == CheckTypeTTL && !check.ExpiresAt.IsZero() && check.ExpiresAt.Before(time.Now()) {
		check.Status = StatusCritical
		check.Output = "TTL expired"
	}

	return check, true
}

func (m *Manager) ListChecks() []*Check {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	checks := make([]*Check, 0, len(m.checks))
	now := time.Now()

	for _, check := range m.checks {
		// Update TTL check status
		if check.Type == CheckTypeTTL && !check.ExpiresAt.IsZero() && check.ExpiresAt.Before(now) {
			check.Status = StatusCritical
			check.Output = "TTL expired"
		}
		checks = append(checks, check)
	}

	return checks
}

func (m *Manager) UpdateTTLCheck(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	check, exists := m.checks[id]
	if !exists {
		return fmt.Errorf("check not found")
	}

	if check.Type != CheckTypeTTL {
		return fmt.Errorf("check is not a TTL check")
	}

	check.Status = StatusPassing
	check.Output = "TTL check passed"
	check.LastCheck = time.Now()
	if check.TTL > 0 {
		check.ExpiresAt = time.Now().Add(check.TTL)
	}

	m.log.Info("TTL check updated",
		logger.String("id", check.ID),
		logger.String("service", check.ServiceID))

	return nil
}

func (m *Manager) RemoveCheck(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.checks[id]; !exists {
		return fmt.Errorf("check not found")
	}

	delete(m.checks, id)

	m.log.Info("Health check removed", logger.String("id", id))
	return nil
}

func (m *Manager) runCheck(check *Check) {
	ticker := time.NewTicker(check.Interval)
	defer ticker.Stop()

	// Run initial check
	m.performCheck(check)

	for {
		select {
		case <-ticker.C:
			m.performCheck(check)
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *Manager) performCheck(check *Check) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if the check still exists (might have been removed)
	if _, exists := m.checks[check.ID]; !exists {
		return
	}

	ctx, cancel := context.WithTimeout(m.ctx, check.Timeout)
	defer cancel()

	var status Status
	var output string
	var err error

	switch check.Type {
	case CheckTypeHTTP:
		status, output, err = m.httpChecker.Check(ctx, check)
	case CheckTypeTCP:
		status, output, err = m.tcpChecker.Check(ctx, check)
	case CheckTypeGRPC:
		status, output, err = m.grpcChecker.Check(ctx, check)
	default:
		status = StatusCritical
		output = fmt.Sprintf("Unknown check type: %s", check.Type)
		err = fmt.Errorf("unknown check type")
	}

	check.Status = status
	check.Output = output
	check.LastCheck = time.Now()

	if err != nil {
		m.log.Warn("Health check failed",
			logger.String("id", check.ID),
			logger.String("name", check.Name),
			logger.String("type", string(check.Type)),
			logger.String("status", string(status)),
			logger.Error(err))
	} else {
		m.log.Debug("Health check completed",
			logger.String("id", check.ID),
			logger.String("name", check.Name),
			logger.String("type", string(check.Type)),
			logger.String("status", string(status)))
	}
}

func (m *Manager) Stop() {
	m.cancel()
	close(m.stopCh)
}
