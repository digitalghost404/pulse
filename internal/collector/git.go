package collector

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/discovery"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

type GitCollector struct{}

func (g *GitCollector) Name() string      { return "git" }
func (g *GitCollector) EnvVars() []string { return nil }

func (g *GitCollector) Enabled(cfg *config.Config) bool {
	return cfg.AdapterEnabled("git")
}

func (g *GitCollector) Collect(ctx context.Context, s store.Store, cfg *config.Config, syncID int64) error {
	repos, err := discovery.FindRepos(cfg.Projects.Scan, cfg.Projects.Ignore)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		snap, err := scanRepo(ctx, repo)
		if err != nil {
			continue // skip repos that fail
		}
		if err := s.SaveGitSnapshot(ctx, syncID, snap); err != nil {
			return err
		}

		branches, err := scanBranches(ctx, repo)
		if err == nil && len(branches) > 0 {
			if err := s.SaveGitBranches(ctx, syncID, branches); err != nil {
				return err
			}
		}
	}

	return nil
}

func scanRepo(ctx context.Context, repo discovery.Repo) (domain.GitSnapshot, error) {
	snap := domain.GitSnapshot{
		RepoPath: repo.Path,
		RepoName: repo.Name,
	}

	// Current branch
	snap.Branch = gitOutput(ctx, repo.Path, "rev-parse", "--abbrev-ref", "HEAD")

	// Dirty files count
	status := gitOutput(ctx, repo.Path, "status", "--porcelain")
	if status != "" {
		snap.DirtyFiles = len(strings.Split(strings.TrimSpace(status), "\n"))
	}

	// Ahead/behind
	revList := gitOutput(ctx, repo.Path, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if revList != "" {
		parts := strings.Fields(revList)
		if len(parts) == 2 {
			snap.Ahead, _ = strconv.Atoi(parts[0])
			snap.Behind, _ = strconv.Atoi(parts[1])
		}
	}

	// Last commit
	snap.LastCommitHash = gitOutput(ctx, repo.Path, "rev-parse", "--short", "HEAD")
	snap.LastCommitMsg = gitOutput(ctx, repo.Path, "log", "-1", "--format=%s")
	dateStr := gitOutput(ctx, repo.Path, "log", "-1", "--format=%aI")
	if dateStr != "" {
		snap.LastCommitAt, _ = time.Parse(time.RFC3339, dateStr)
	}

	return snap, nil
}

func scanBranches(ctx context.Context, repo discovery.Repo) ([]domain.GitBranch, error) {
	// Get all local branches
	output := gitOutput(ctx, repo.Path, "for-each-ref", "--format=%(refname:short)\t%(committerdate:iso-strict)\t%(HEAD)", "refs/heads/")
	if output == "" {
		return nil, nil
	}

	// Get merged branches once (avoid O(n) git calls)
	mergedSet := make(map[string]bool)
	mergedOutput := gitOutput(ctx, repo.Path, "branch", "--merged", "HEAD")
	for _, line := range strings.Split(mergedOutput, "\n") {
		name := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "*"))
		if name != "" {
			mergedSet[name] = true
		}
	}

	var branches []domain.GitBranch
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		b := domain.GitBranch{
			RepoPath:   repo.Path,
			BranchName: parts[0],
			IsCurrent:  strings.TrimSpace(parts[2]) == "*",
			IsMerged:   mergedSet[parts[0]],
		}
		if parts[1] != "" {
			b.LastCommitAt, _ = time.Parse(time.RFC3339, parts[1])
		}

		branches = append(branches, b)
	}

	return branches, nil
}

func gitOutput(ctx context.Context, dir string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func init() {
	Register(&GitCollector{})
}
