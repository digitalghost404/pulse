package sync_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
	psync "github.com/xcoleman/pulse/internal/sync"
)

type fakeCollector struct {
	name   string
	err    error
	called bool
}

func (f *fakeCollector) Name() string                                                              { return f.name }
func (f *fakeCollector) EnvVars() []string                                                         { return nil }
func (f *fakeCollector) Enabled(cfg *config.Config) bool                                           { return true }
func (f *fakeCollector) Collect(ctx context.Context, s store.Store, cfg *config.Config, syncID int64) error {
	f.called = true
	return f.err
}

func newTestStoreForSync(t *testing.T) store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSyncEngine_AllSuccess(t *testing.T) {
	s := newTestStoreForSync(t)
	cfg := &config.Config{Sync: config.SyncConfig{Timeout: "30s"}}

	c1 := &fakeCollector{name: "fake1"}
	c2 := &fakeCollector{name: "fake2"}

	engine := psync.NewEngine(s, cfg)
	result := engine.Run(context.Background(), []collector.Collector{c1, c2})

	if result.Status != "success" {
		t.Errorf("expected success, got %s", result.Status)
	}
	if !c1.called || !c2.called {
		t.Error("expected both collectors to be called")
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %v", result.Errors)
	}
}

func TestSyncEngine_PartialFailure(t *testing.T) {
	s := newTestStoreForSync(t)
	cfg := &config.Config{Sync: config.SyncConfig{Timeout: "30s"}}

	c1 := &fakeCollector{name: "ok"}
	c2 := &fakeCollector{name: "fail", err: errors.New("boom")}

	engine := psync.NewEngine(s, cfg)
	result := engine.Run(context.Background(), []collector.Collector{c1, c2})

	if result.Status != "partial" {
		t.Errorf("expected partial, got %s", result.Status)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestSyncEngine_TotalFailure(t *testing.T) {
	s := newTestStoreForSync(t)
	cfg := &config.Config{Sync: config.SyncConfig{Timeout: "30s"}}

	c1 := &fakeCollector{name: "fail1", err: errors.New("boom1")}
	c2 := &fakeCollector{name: "fail2", err: errors.New("boom2")}

	engine := psync.NewEngine(s, cfg)
	result := engine.Run(context.Background(), []collector.Collector{c1, c2})

	if result.Status != "failed" {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if len(result.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors))
	}
}

func TestSyncEngine_OnlyMode(t *testing.T) {
	s := newTestStoreForSync(t)
	cfg := &config.Config{Sync: config.SyncConfig{Timeout: "30s"}}

	c1 := &fakeCollector{name: "git"}
	c2 := &fakeCollector{name: "docker"}

	engine := psync.NewEngine(s, cfg)
	result := engine.RunOnly(context.Background(), []collector.Collector{c1, c2}, "git")

	if result.Status != "success" {
		t.Errorf("expected success, got %s", result.Status)
	}
	if !c1.called {
		t.Error("expected git collector to be called")
	}
	if c2.called {
		t.Error("expected docker collector NOT to be called")
	}
}
