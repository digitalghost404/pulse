package collector_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
)

// createTestRepo initializes a git repo with one commit.
func createTestRepo(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	os.MkdirAll(dir, 0755)

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "checkout", "-b", "main"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("command %v failed: %s: %v", args, out, err)
		}
	}

	// Create a file and commit
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644)
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit failed: %s: %v", out, err)
	}

	return dir
}

func newTestStoreForCollector(t *testing.T) store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestGitCollector_Collect(t *testing.T) {
	repoDir := createTestRepo(t, "test-repo")

	s := newTestStoreForCollector(t)
	ctx := context.Background()
	syncID, _ := s.CreateSyncRun(ctx)

	cfg := &config.Config{
		Projects: config.ProjectsConfig{
			Scan: []string{filepath.Dir(repoDir)},
		},
	}

	gc := &collector.GitCollector{}
	err := gc.Collect(ctx, s, cfg, syncID)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	snapshots, err := s.GetGitSnapshots(ctx, syncID)
	if err != nil {
		t.Fatalf("GetGitSnapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
	}

	snap := snapshots[0]
	if snap.RepoName != "test-repo" {
		t.Errorf("expected repo name test-repo, got %s", snap.RepoName)
	}
	if snap.Branch != "main" {
		t.Errorf("expected branch main, got %s", snap.Branch)
	}
	if snap.DirtyFiles != 0 {
		t.Errorf("expected 0 dirty files, got %d", snap.DirtyFiles)
	}
}

func TestGitCollector_DirtyFiles(t *testing.T) {
	repoDir := createTestRepo(t, "dirty-repo")

	// Create an uncommitted file
	os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("dirty"), 0644)

	s := newTestStoreForCollector(t)
	ctx := context.Background()
	syncID, _ := s.CreateSyncRun(ctx)

	cfg := &config.Config{
		Projects: config.ProjectsConfig{
			Scan: []string{filepath.Dir(repoDir)},
		},
	}

	gc := &collector.GitCollector{}
	err := gc.Collect(ctx, s, cfg, syncID)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	snapshots, _ := s.GetGitSnapshots(ctx, syncID)
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
	}
	if snapshots[0].DirtyFiles != 1 {
		t.Errorf("expected 1 dirty file, got %d", snapshots[0].DirtyFiles)
	}
}

func TestGitCollector_EnabledCheck(t *testing.T) {
	gc := &collector.GitCollector{}

	// Enabled by default
	cfg := &config.Config{}
	if !gc.Enabled(cfg) {
		t.Error("expected git collector to be enabled by default")
	}

	// Disabled in config
	cfg = &config.Config{Adapters: map[string]bool{"git": false}}
	if gc.Enabled(cfg) {
		t.Error("expected git collector to be disabled")
	}
}
