// Package sync orchestrates collector execution and data persistence.
package sync

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
)

type Result struct {
	SyncID int64
	Status string // success, partial, failed
	Errors []string
}

type Engine struct {
	store store.Store
	cfg   *config.Config
}

func NewEngine(s store.Store, cfg *config.Config) *Engine {
	return &Engine{store: s, cfg: cfg}
}

func (e *Engine) Run(ctx context.Context, collectors []collector.Collector) Result {
	return e.runCollectors(ctx, collectors)
}

func (e *Engine) RunOnly(ctx context.Context, collectors []collector.Collector, only string) Result {
	var filtered []collector.Collector
	for _, c := range collectors {
		if c.Name() == only {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return Result{Status: "failed", Errors: []string{fmt.Sprintf("collector %q not found", only)}}
	}
	return e.runCollectors(ctx, filtered)
}

func (e *Engine) runCollectors(ctx context.Context, collectors []collector.Collector) Result {
	syncID, err := e.store.CreateSyncRun(ctx)
	if err != nil {
		return Result{Status: "failed", Errors: []string{fmt.Sprintf("creating sync run: %v", err)}}
	}

	timeout := parseDuration(e.cfg.Sync.Timeout, 30*time.Second)

	var errs []string
	for _, c := range collectors {
		cCtx, cancel := context.WithTimeout(ctx, timeout)
		if err := c.Collect(cCtx, e.store, e.cfg, syncID); err != nil {
			log.Printf("WARN: collector %q failed: %v", c.Name(), err)
			errs = append(errs, fmt.Sprintf("%s: %v", c.Name(), err))
		}
		cancel()
	}

	// Prune old briefing history (30 days)
	e.store.PruneBriefingHistory(ctx, time.Now().Add(-30*24*time.Hour))

	var status string
	switch {
	case len(errs) == 0:
		status = "success"
	case len(errs) < len(collectors):
		status = "partial"
	default:
		status = "failed"
	}

	errMsg := strings.Join(errs, "; ")
	e.store.CompleteSyncRun(ctx, syncID, status, errMsg)

	return Result{SyncID: syncID, Status: status, Errors: errs}
}

func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}
