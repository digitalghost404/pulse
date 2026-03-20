package briefing_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/xcoleman/pulse/internal/briefing"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

func seedTestStore(t *testing.T) (store.Store, int64) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	ctx := context.Background()
	syncID, _ := s.CreateSyncRun(ctx)

	s.SaveGitSnapshot(ctx, syncID, domain.GitSnapshot{
		RepoPath: "/projects/pulse", RepoName: "pulse", Branch: "main",
		DirtyFiles: 0, Ahead: 0, Behind: 0,
		LastCommitHash: "abc123", LastCommitMsg: "initial", LastCommitAt: time.Now(),
	})
	s.SaveGitSnapshot(ctx, syncID, domain.GitSnapshot{
		RepoPath: "/projects/cortex", RepoName: "cortex", Branch: "main",
		DirtyFiles: 3, Ahead: 2, Behind: 0,
		LastCommitHash: "def456", LastCommitMsg: "fix bug", LastCommitAt: time.Now(),
	})

	s.SaveGitHubNotifications(ctx, syncID, []domain.Notification{
		{RepoName: "obsidian-mcp", Type: "pr", Title: "Fix indexing", URL: "https://github.com/pr/42", State: "open", UpdatedAt: time.Now()},
	})

	s.SaveCostEntry(ctx, syncID, domain.CostEntry{
		Service: "claude", PeriodStart: time.Now().Add(-24 * time.Hour), PeriodEnd: time.Now(),
		AmountCents: 1482, Currency: "USD", UsageQuantity: 150000, UsageUnit: "tokens",
	})

	s.SaveSystemSnapshot(ctx, syncID, domain.SystemSnapshot{
		CPUPct: 12.5, MemoryUsedMB: 18200, MemoryTotalMB: 32000, DiskUsedGB: 142, DiskTotalGB: 256,
	})

	s.CompleteSyncRun(ctx, syncID, "success", "")
	return s, syncID
}

func TestBuildBriefing(t *testing.T) {
	s, _ := seedTestStore(t)
	cfg := &config.Config{Costs: config.CostsConfig{DefaultPeriod: "30d", Currency: "USD"}}

	engine := briefing.NewEngine(s, cfg)
	b, err := engine.Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(b.Projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(b.Projects))
	}
	if len(b.Notifications) != 1 {
		t.Errorf("expected 1 notification, got %d", len(b.Notifications))
	}
	if b.CostSummary.TotalCents != 1482 {
		t.Errorf("expected 1482 total cents, got %d", b.CostSummary.TotalCents)
	}
	if b.System.CPUPct != 12.5 {
		t.Errorf("expected CPU 12.5, got %f", b.System.CPUPct)
	}
}

func TestParsePeriod(t *testing.T) {
	tests := []struct {
		period string
		days   int
	}{
		{"30d", 30},
		{"7d", 7},
		{"1d", 1},
		{"", 30},      // default
		{"bad", 30},   // fallback
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			result := briefing.ParsePeriod(tt.period)
			expected := time.Now().Add(-time.Duration(tt.days) * 24 * time.Hour)
			diff := result.Sub(expected)
			if diff < -time.Second || diff > time.Second {
				t.Errorf("ParsePeriod(%q): expected ~%d days ago, got diff %v", tt.period, tt.days, diff)
			}
		})
	}
}

func TestBuildWithOptions_Since(t *testing.T) {
	s, _ := seedTestStore(t)
	cfg := &config.Config{Costs: config.CostsConfig{DefaultPeriod: "30d", Currency: "USD"}}

	engine := briefing.NewEngine(s, cfg)

	// Build with explicit since
	since := time.Now().Add(-7 * 24 * time.Hour)
	b, err := engine.BuildWithOptions(context.Background(), briefing.BuildOptions{Since: since})
	if err != nil {
		t.Fatalf("BuildWithOptions: %v", err)
	}

	// Should still return data (since is within the cost entry range)
	if b.CostSummary.TotalCents != 1482 {
		t.Errorf("expected 1482 total cents, got %d", b.CostSummary.TotalCents)
	}
}

func TestBuildBriefing_EmptyDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, _ := store.NewSQLite(dbPath)
	defer s.Close()

	cfg := &config.Config{Costs: config.CostsConfig{DefaultPeriod: "30d", Currency: "USD"}}
	engine := briefing.NewEngine(s, cfg)
	_, err := engine.Build(context.Background())

	if err == nil {
		t.Error("expected error for empty DB")
	}
}
