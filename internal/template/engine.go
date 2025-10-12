package template

import (
	"context"
	"fmt"
	"sync"

	"github.com/neogan74/konsul/internal/logger"
)

// Engine is the main template engine orchestrator
type Engine struct {
	config   Config
	renderer *Renderer
	watchers []*Watcher
	log      logger.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// New creates a new template engine
func New(config Config, kvStore KVStoreReader, serviceStore ServiceStoreReader, log logger.Logger) *Engine {
	ctx, cancel := context.WithCancel(context.Background())

	renderCtx := &RenderContext{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
		DryRun:       config.DryRun,
	}

	return &Engine{
		config:   config,
		renderer: NewRenderer(renderCtx),
		log:      log,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// RunOnce renders all templates once and exits
func (e *Engine) RunOnce() error {
	e.log.Info("Running template engine in once mode",
		logger.Int("templates", len(e.config.Templates)))

	var errs []error
	for _, tmpl := range e.config.Templates {
		result, err := e.renderer.Render(tmpl)
		if err != nil {
			e.log.Error("Failed to render template",
				logger.String("source", tmpl.Source),
				logger.Error(err))
			errs = append(errs, err)
			continue
		}

		e.logResult(result)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to render %d templates", len(errs))
	}

	return nil
}

// Run starts the template engine in watch mode
func (e *Engine) Run(ctx context.Context) error {
	e.log.Info("Starting template engine in watch mode",
		logger.Int("templates", len(e.config.Templates)))

	// Render templates once at startup
	if err := e.RunOnce(); err != nil {
		e.log.Warn("Initial template render had errors", logger.Error(err))
	}

	// Start watchers for each template
	for _, tmpl := range e.config.Templates {
		watcher := NewWatcher(e, tmpl)
		e.watchers = append(e.watchers, watcher)

		e.wg.Add(1)
		go func(w *Watcher) {
			defer e.wg.Done()
			w.Watch(ctx)
		}(watcher)
	}

	// Wait for context cancellation
	<-ctx.Done()

	e.log.Info("Shutting down template engine")
	e.cancel()
	e.wg.Wait()

	return nil
}

// Stop stops the template engine
func (e *Engine) Stop() {
	e.cancel()
	e.wg.Wait()
}

// RenderTemplate renders a single template
func (e *Engine) RenderTemplate(tmpl TemplateConfig) (*RenderResult, error) {
	return e.renderer.Render(tmpl)
}

// logResult logs the result of a template render
func (e *Engine) logResult(result *RenderResult) {
	fields := []logger.Field{
		logger.String("source", result.Template.Source),
		logger.String("destination", result.Template.Destination),
		logger.Duration("duration", result.Duration),
	}

	if result.Error != nil {
		e.log.Error("Template render failed", append(fields, logger.Error(result.Error))...)
		return
	}

	if e.config.DryRun {
		e.log.Info("Template rendered (dry-run)",
			append(fields,
				logger.Int("content_size", len(result.Content)))...)
		return
	}

	if result.Written {
		fields = append(fields, logger.String("written", "true"))
	}

	if result.CommandExecuted {
		fields = append(fields,
			logger.String("command_executed", "true"),
			logger.String("command", result.Template.Command))
	}

	e.log.Info("Template rendered successfully", fields...)
}
