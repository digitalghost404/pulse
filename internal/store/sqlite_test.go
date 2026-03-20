package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

func newTestStore(t *testing.T) store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSyncRunLifecycle(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	id, err := s.CreateSyncRun(ctx)
	if err != nil {
		t.Fatalf("CreateSyncRun: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive ID, got %d", id)
	}

	err = s.CompleteSyncRun(ctx, id, "success", "")
	if err != nil {
		t.Fatalf("CompleteSyncRun: %v", err)
	}

	latestID, err := s.LatestSyncID(ctx)
	if err != nil {
		t.Fatalf("LatestSyncID: %v", err)
	}
	if latestID != id {
		t.Errorf("expected latest sync ID %d, got %d", id, latestID)
	}
}

func TestGitSnapshotRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	syncID, _ := s.CreateSyncRun(ctx)

	snapshot := domain.GitSnapshot{
		RepoPath:       "/home/user/projects/pulse",
		RepoName:       "pulse",
		Branch:         "main",
		DirtyFiles:     3,
		Ahead:          2,
		Behind:         0,
		LastCommitHash: "abc123",
		LastCommitMsg:  "feat: initial commit",
		LastCommitAt:   time.Now().Truncate(time.Second),
	}

	if err := s.SaveGitSnapshot(ctx, syncID, snapshot); err != nil {
		t.Fatalf("SaveGitSnapshot: %v", err)
	}

	snapshots, err := s.GetGitSnapshots(ctx, syncID)
	if err != nil {
		t.Fatalf("GetGitSnapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
	}

	got := snapshots[0]
	if got.RepoName != "pulse" {
		t.Errorf("expected repo name pulse, got %s", got.RepoName)
	}
	if got.DirtyFiles != 3 {
		t.Errorf("expected 3 dirty files, got %d", got.DirtyFiles)
	}
	if got.Ahead != 2 {
		t.Errorf("expected ahead 2, got %d", got.Ahead)
	}
}

func TestCostEntryRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	syncID, _ := s.CreateSyncRun(ctx)

	now := time.Now().Truncate(time.Second)
	entry := domain.CostEntry{
		Service:       "claude",
		PeriodStart:   now.Add(-24 * time.Hour),
		PeriodEnd:     now,
		AmountCents:   1482,
		Currency:      "USD",
		UsageQuantity: 150000,
		UsageUnit:     "tokens",
		RawData:       `{"model":"opus"}`,
	}

	if err := s.SaveCostEntry(ctx, syncID, entry); err != nil {
		t.Fatalf("SaveCostEntry: %v", err)
	}

	entries, err := s.GetCostEntries(ctx, now.Add(-48*time.Hour))
	if err != nil {
		t.Fatalf("GetCostEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.Service != "claude" {
		t.Errorf("expected service claude, got %s", got.Service)
	}
	if got.AmountCents != 1482 {
		t.Errorf("expected 1482 cents, got %d", got.AmountCents)
	}
}

func TestBriefingHistoryPrune(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Save an old briefing and a new one
	old := domain.BriefingEntry{
		CreatedAt: time.Now().Add(-60 * 24 * time.Hour),
		Content:   "old briefing",
		Writer:    "stdout",
	}
	recent := domain.BriefingEntry{
		CreatedAt: time.Now(),
		Content:   "recent briefing",
		Writer:    "stdout",
	}

	if err := s.SaveBriefing(ctx, old); err != nil {
		t.Fatalf("SaveBriefing (old): %v", err)
	}
	if err := s.SaveBriefing(ctx, recent); err != nil {
		t.Fatalf("SaveBriefing (recent): %v", err)
	}

	pruned, err := s.PruneBriefingHistory(ctx, time.Now().Add(-30*24*time.Hour))
	if err != nil {
		t.Fatalf("PruneBriefingHistory: %v", err)
	}
	if pruned != 1 {
		t.Errorf("expected 1 pruned, got %d", pruned)
	}
}

func TestDockerSnapshotRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	syncID, _ := s.CreateSyncRun(ctx)

	snap := domain.DockerSnapshot{
		ContainerName: "redis",
		Image:         "redis:7",
		Status:        "running",
		Ports:         `["6379:6379"]`,
		CPUPct:        1.2,
		MemoryMB:      64.5,
	}

	if err := s.SaveDockerSnapshot(ctx, syncID, snap); err != nil {
		t.Fatalf("SaveDockerSnapshot: %v", err)
	}

	snaps, err := s.GetDockerSnapshots(ctx, syncID)
	if err != nil {
		t.Fatalf("GetDockerSnapshots: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}
	if snaps[0].ContainerName != "redis" {
		t.Errorf("expected container name redis, got %s", snaps[0].ContainerName)
	}
}

func TestSystemSnapshotRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	syncID, _ := s.CreateSyncRun(ctx)

	snap := domain.SystemSnapshot{
		CPUPct:        12.5,
		MemoryUsedMB:  18200,
		MemoryTotalMB: 32000,
		DiskUsedGB:    142.3,
		DiskTotalGB:   256.0,
	}

	if err := s.SaveSystemSnapshot(ctx, syncID, snap); err != nil {
		t.Fatalf("SaveSystemSnapshot: %v", err)
	}

	got, err := s.GetSystemSnapshot(ctx, syncID)
	if err != nil {
		t.Fatalf("GetSystemSnapshot: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if got.CPUPct != 12.5 {
		t.Errorf("expected CPU 12.5, got %f", got.CPUPct)
	}
}

func TestGitHubNotificationsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	syncID, _ := s.CreateSyncRun(ctx)

	notifs := []domain.Notification{
		{
			RepoName:  "obsidian-mcp",
			Type:      "pr",
			Title:     "Fix FTS5 indexing",
			URL:       "https://github.com/xcoleman/obsidian-mcp/pull/42",
			State:     "open",
			UpdatedAt: time.Now().Truncate(time.Second),
		},
	}

	if err := s.SaveGitHubNotifications(ctx, syncID, notifs); err != nil {
		t.Fatalf("SaveGitHubNotifications: %v", err)
	}

	got, err := s.GetGitHubNotifications(ctx, syncID)
	if err != nil {
		t.Fatalf("GetGitHubNotifications: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(got))
	}
	if got[0].Title != "Fix FTS5 indexing" {
		t.Errorf("expected title 'Fix FTS5 indexing', got %s", got[0].Title)
	}
}

func TestGetLastBriefingTime(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// No briefings — should return zero time
	bt, err := s.GetLastBriefingTime(ctx)
	if err != nil {
		t.Fatalf("GetLastBriefingTime: %v", err)
	}
	if !bt.IsZero() {
		t.Errorf("expected zero time, got %v", bt)
	}

	// Add a briefing
	entry := domain.BriefingEntry{
		CreatedAt: time.Now().Truncate(time.Second),
		Content:   "test",
		Writer:    "stdout",
	}
	if err := s.SaveBriefing(ctx, entry); err != nil {
		t.Fatal(err)
	}

	bt, err = s.GetLastBriefingTime(ctx)
	if err != nil {
		t.Fatalf("GetLastBriefingTime: %v", err)
	}
	if bt.IsZero() {
		t.Error("expected non-zero time after saving briefing")
	}
}
