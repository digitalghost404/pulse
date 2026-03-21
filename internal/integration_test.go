package internal_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/xcoleman/pulse/internal/briefing"
	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
	psync "github.com/xcoleman/pulse/internal/sync"
	"github.com/xcoleman/pulse/internal/writer"
)

func TestFullSyncAndBriefing(t *testing.T) {
	// Setup: create a temp repo
	repoDir := filepath.Join(t.TempDir(), "test-repo")
	os.MkdirAll(repoDir, 0755)
	for _, args := range [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "checkout", "-b", "main"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		cmd.Run()
	}
	os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Test"), 0644)
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = repoDir
	cmd.Run()

	// Setup: DB and config
	dbPath := filepath.Join(t.TempDir(), "pulse.db")
	s, err := store.NewSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	cfg := &config.Config{
		Projects: config.ProjectsConfig{
			Scan: []string{filepath.Dir(repoDir)},
		},
		Sync:  config.SyncConfig{Timeout: "30s"},
		Costs: config.CostsConfig{DefaultPeriod: "30d", Currency: "USD"},
	}

	// Step 1: Sync (git only — no GitHub token, no Docker needed)
	gitCollector, ok := collector.Get("git")
	if !ok {
		t.Fatal("git collector not registered")
	}

	engine := psync.NewEngine(s, cfg)
	result := engine.Run(context.Background(), []collector.Collector{gitCollector})
	if result.Status == "failed" {
		t.Fatalf("sync failed: %v", result.Errors)
	}

	// Step 2: Build briefing
	bEngine := briefing.NewEngine(s, cfg)
	b, err := bEngine.Build(context.Background())
	if err != nil {
		t.Fatalf("build briefing: %v", err)
	}

	if len(b.Projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(b.Projects))
	}
	if b.Projects[0].RepoName != "test-repo" {
		t.Errorf("expected test-repo, got %s", b.Projects[0].RepoName)
	}

	// Step 3: Write to stdout
	var buf bytes.Buffer
	w := writer.NewStdoutWriter(&buf)
	if err := w.Write(context.Background(), b, cfg); err != nil {
		t.Fatalf("stdout write: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("test-repo")) {
		t.Error("expected output to contain 'test-repo'")
	}

	// Step 4: Save briefing history and verify time tracking
	s.SaveBriefing(context.Background(), domain.BriefingEntry{
		CreatedAt: time.Now(),
		Content:   output,
		Writer:    "stdout",
	})

	lastTime, err := s.GetLastBriefingTime(context.Background())
	if err != nil {
		t.Fatalf("GetLastBriefingTime: %v", err)
	}
	if lastTime.IsZero() {
		t.Error("expected non-zero last briefing time after save")
	}

	// Step 5: Build again with options — should use last briefing time
	b2, err := bEngine.BuildWithOptions(context.Background(), briefing.BuildOptions{})
	if err != nil {
		t.Fatalf("BuildWithOptions: %v", err)
	}
	if len(b2.Projects) != 1 {
		t.Errorf("expected 1 project on second build, got %d", len(b2.Projects))
	}
}
