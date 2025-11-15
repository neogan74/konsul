package audit

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/metrics"
)

var (
	// ErrManagerClosed is returned when writes occur after shutdown.
	ErrManagerClosed = errors.New("audit manager closed")
	// ErrNilEvent is returned when callers attempt to record a nil event.
	ErrNilEvent = errors.New("audit event is nil")
)

// DropPolicy determines how the manager handles a full channel.
type DropPolicy string

const (
	DropPolicyDrop  DropPolicy = "drop"
	DropPolicyBlock DropPolicy = "block"
)

// Config mirrors the public audit configuration.
type Config struct {
	Enabled       bool
	Sink          string
	FilePath      string
	BufferSize    int
	FlushInterval time.Duration
	DropPolicy    DropPolicy
}

// Writer defines the sink contract for audit events.
type Writer interface {
	Write(event *Event) error
	Flush() error
	Close(ctx context.Context) error
}

// Manager handles buffering and delivery of audit events.
type Manager struct {
	cfg    Config
	log    logger.Logger
	writer Writer

	events chan *Event
	wg     sync.WaitGroup

	flushTicker *time.Ticker
	stopOnce    sync.Once

	enabled bool
	closed  bool
	mu      sync.RWMutex
}

// NewManager builds a new audit manager. When disabled, it falls back to a no-op manager.
func NewManager(cfg Config, log logger.Logger) (*Manager, error) {
	if log == nil {
		log = logger.GetDefault()
	}

	if !cfg.Enabled {
		return &Manager{
			cfg:     cfg,
			log:     log,
			enabled: false,
		}, nil
	}

	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1024
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = time.Second
	}
	if cfg.DropPolicy == "" {
		cfg.DropPolicy = DropPolicyDrop
	}

	writer, err := newWriter(cfg)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		cfg:         cfg,
		log:         log,
		writer:      writer,
		events:      make(chan *Event, cfg.BufferSize),
		flushTicker: time.NewTicker(cfg.FlushInterval),
		enabled:     true,
	}

	m.wg.Add(1)
	go m.run()

	return m, nil
}

// Enabled indicates whether audit logging is active.
func (m *Manager) Enabled() bool {
	return m != nil && m.enabled
}

// Record buffers an audit event for asynchronous delivery.
func (m *Manager) Record(ctx context.Context, event *Event) (string, error) {
	if m == nil {
		return "", nil
	}
	if !m.enabled {
		return "", nil
	}
	if event == nil {
		return "", ErrNilEvent
	}

	m.mu.RLock()
	closed := m.closed
	m.mu.RUnlock()
	if closed {
		metrics.AuditEventsDroppedTotal.WithLabelValues(m.cfg.Sink, "manager_closed").Inc()
		return "", ErrManagerClosed
	}

	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Metadata == nil {
		event.Metadata = map[string]string{}
	}

	select {
	case m.events <- event:
		return event.ID, nil
	default:
		if m.cfg.DropPolicy == DropPolicyDrop {
			metrics.AuditEventsDroppedTotal.WithLabelValues(m.cfg.Sink, "buffer_full").Inc()
			return "", errors.New("audit buffer full")
		}
		select {
		case m.events <- event:
			return event.ID, nil
		case <-ctx.Done():
			metrics.AuditEventsDroppedTotal.WithLabelValues(m.cfg.Sink, "context_cancelled").Inc()
			return "", ctx.Err()
		}
	}
}

func (m *Manager) run() {
	defer m.wg.Done()

	for {
		select {
		case event, ok := <-m.events:
			if !ok {
				m.flush()
				return
			}
			m.write(event)
		case <-m.flushTicker.C:
			m.flush()
		}
	}
}

// Shutdown drains the buffer, flushes the writer, and closes resources.
func (m *Manager) Shutdown(ctx context.Context) error {
	if m == nil || !m.enabled {
		return nil
	}

	m.stopOnce.Do(func() {
		m.mu.Lock()
		m.closed = true
		m.mu.Unlock()
		close(m.events)
	})

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	m.flushTicker.Stop()
	m.flush()
	return m.writer.Close(ctx)
}

func (m *Manager) write(event *Event) {
	if event == nil {
		return
	}
	if err := m.writer.Write(event); err != nil {
		m.log.Error("Failed to write audit event", logger.Error(err))
		metrics.AuditEventsTotal.WithLabelValues(m.cfg.Sink, "error").Inc()
		return
	}
	metrics.AuditEventsTotal.WithLabelValues(m.cfg.Sink, "written").Inc()
}

func (m *Manager) flush() {
	start := time.Now()
	if err := m.writer.Flush(); err != nil {
		m.log.Error("Failed to flush audit writer", logger.Error(err))
		return
	}
	metrics.AuditWriterFlushDuration.WithLabelValues(m.cfg.Sink).Observe(time.Since(start).Seconds())
}
