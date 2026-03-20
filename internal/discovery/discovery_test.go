package discovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xcoleman/pulse/internal/discovery"
)

func setupRepos(t *testing.T, names ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range names {
		gitDir := filepath.Join(root, name, ".git")
		if err := os.MkdirAll(gitDir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestFindRepos_Basic(t *testing.T) {
	root := setupRepos(t, "project-a", "project-b")

	repos, err := discovery.FindRepos([]string{root}, nil)
	if err != nil {
		t.Fatalf("FindRepos: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(repos))
	}
}

func TestFindRepos_IgnoreList(t *testing.T) {
	root := setupRepos(t, "project-a", "vendor-stuff")

	repos, err := discovery.FindRepos([]string{root}, []string{"vendor-stuff"})
	if err != nil {
		t.Fatalf("FindRepos: %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("expected 1 repo, got %d", len(repos))
	}
	if repos[0].Name != "project-a" {
		t.Errorf("expected project-a, got %s", repos[0].Name)
	}
}

func TestFindRepos_MaxDepth(t *testing.T) {
	root := t.TempDir()
	// Depth 1 — should be found
	os.MkdirAll(filepath.Join(root, "project-a", ".git"), 0755)
	// Depth 3 — should NOT be found (too deep)
	os.MkdirAll(filepath.Join(root, "deep", "nested", "project-b", ".git"), 0755)

	repos, err := discovery.FindRepos([]string{root}, nil)
	if err != nil {
		t.Fatalf("FindRepos: %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("expected 1 repo (depth limit), got %d", len(repos))
	}
}

func TestFindRepos_DefaultExclusions(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "project-a", ".git"), 0755)
	os.MkdirAll(filepath.Join(root, "project-a", "node_modules", "dep", ".git"), 0755)

	repos, err := discovery.FindRepos([]string{root}, nil)
	if err != nil {
		t.Fatalf("FindRepos: %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("expected 1 repo (node_modules excluded), got %d", len(repos))
	}
}

func TestFindRepos_MultipleScanDirs(t *testing.T) {
	root1 := setupRepos(t, "repo1")
	root2 := setupRepos(t, "repo2")

	repos, err := discovery.FindRepos([]string{root1, root2}, nil)
	if err != nil {
		t.Fatalf("FindRepos: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(repos))
	}
}
