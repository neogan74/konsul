package template

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"go.uber.org/zap"
)

// Watcher watches for changes and triggers template re-renders
type Watcher struct {
	engine        *Engine
	template      TemplateConfig
	lastHash      string
	lastRender    time.Time
	pendingRender bool
	minWait       time.Duration
	maxWait       time.Duration
}

// NewWatcher creates a new watcher for a template
func NewWatcher(engine *Engine, template TemplateConfig) *Watcher {
	minWait := 2 * time.Second
	maxWait := 10 * time.Second

	// Use template-specific wait config if available
	if template.Wait != nil {
		minWait = template.Wait.Min
		maxWait = template.Wait.Max
	} else if engine.config.Wait != nil {
		// Fall back to global wait config
		minWait = engine.config.Wait.Min
		maxWait = engine.config.Wait.Max
	}

	return &Watcher{
		engine:   engine,
		template: template,
		minWait:  minWait,
		maxWait:  maxWait,
	}
}

// Watch starts watching for changes
func (w *Watcher) Watch(ctx context.Context) {
	ticker := time.NewTicker(w.minWait)
	defer ticker.Stop()

	maxWaitTimer := time.NewTimer(w.maxWait)
	defer maxWaitTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			// Check if content has changed
			if w.hasChanged() {
				if !w.pendingRender {
					// First change detected, start waiting
					w.pendingRender = true
					maxWaitTimer.Reset(w.maxWait)
				}
			}

		case <-maxWaitTimer.C:
			// Max wait time reached, render if there's a pending change
			if w.pendingRender {
				w.render()
				w.pendingRender = false
			}
			maxWaitTimer.Reset(w.maxWait)
		}
	}
}

// hasChanged checks if the rendered content would be different
func (w *Watcher) hasChanged() bool {
	// Render to a temporary buffer to compute hash
	result, err := w.engine.RenderTemplate(w.template)
	if err != nil {
		// If render fails, consider it unchanged to avoid error loops
		return false
	}

	// Compute hash of rendered content
	hash := computeHash(result.Content)

	// Check if hash has changed
	if hash != w.lastHash {
		w.lastHash = hash
		return true
	}

	return false
}

// render triggers a template re-render
func (w *Watcher) render() {
	// Respect minimum wait time between renders
	if time.Since(w.lastRender) < w.minWait {
		return
	}

	result, err := w.engine.RenderTemplate(w.template)
	if err != nil {
		w.engine.log.Error("Failed to render template in watch mode",
			zap.String("template", w.template.Source),
			zap.Error(err))
		return
	}

	w.engine.logResult(result)
	w.lastRender = time.Now()
}

// computeHash computes SHA256 hash of content
func computeHash(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}
