package collector_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
)

func TestSystemCollector_Collect(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	syncID, _ := s.CreateSyncRun(ctx)

	sc := &collector.SystemCollector{}
	err = sc.Collect(ctx, s, &config.Config{}, syncID)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	snap, err := s.GetSystemSnapshot(ctx, syncID)
	if err != nil {
		t.Fatalf("GetSystemSnapshot: %v", err)
	}
	if snap == nil {
		t.Fatal("expected non-nil system snapshot")
	}

	// Basic sanity — values should be positive on any real system
	if snap.MemoryTotalMB <= 0 {
		t.Errorf("expected positive total memory, got %f", snap.MemoryTotalMB)
	}
	if snap.DiskTotalGB <= 0 {
		t.Errorf("expected positive total disk, got %f", snap.DiskTotalGB)
	}
}

func TestSystemCollector_EnabledCheck(t *testing.T) {
	sc := &collector.SystemCollector{}

	cfg := &config.Config{}
	if !sc.Enabled(cfg) {
		t.Error("expected system collector enabled by default")
	}

	cfg = &config.Config{Adapters: map[string]bool{"system": false}}
	if sc.Enabled(cfg) {
		t.Error("expected system collector disabled")
	}
}
