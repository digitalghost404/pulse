// Package discovery scans the filesystem for project directories.
package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultExclusions are directory names skipped during scanning.
var DefaultExclusions = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	".cache":       true,
	"__pycache__":  true,
}

const maxDepth = 2

// Repo represents a discovered git repository.
type Repo struct {
	Path string // absolute path
	Name string // directory name
}

// FindRepos scans directories for git repos up to maxDepth levels deep.
func FindRepos(scanDirs []string, ignore []string) ([]Repo, error) {
	ignoreSet := make(map[string]bool)
	for _, name := range ignore {
		ignoreSet[name] = true
	}

	var repos []Repo
	seen := make(map[string]bool)

	for _, scanDir := range scanDirs {
		// Expand ~ to home directory
		if strings.HasPrefix(scanDir, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			scanDir = filepath.Join(home, scanDir[2:])
		}

		scanDir, err := filepath.Abs(scanDir)
		if err != nil {
			return nil, err
		}

		found, err := scanDirRecursive(scanDir, ignoreSet, 0)
		if err != nil {
			return nil, err
		}

		for _, r := range found {
			if !seen[r.Path] {
				seen[r.Path] = true
				repos = append(repos, r)
			}
		}
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	return repos, nil
}

func scanDirRecursive(dir string, ignore map[string]bool, depth int) ([]Repo, error) {
	if depth >= maxDepth {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil // skip unreadable dirs
	}

	var repos []Repo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip hidden dirs (except we look for .git inside)
		if name[0] == '.' {
			continue
		}

		// Skip default exclusions and user ignore list
		if DefaultExclusions[name] || ignore[name] {
			continue
		}

		entryPath := filepath.Join(dir, name)

		// Check if this directory is a git repo
		gitDir := filepath.Join(entryPath, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			repos = append(repos, Repo{
				Path: entryPath,
				Name: name,
			})
			continue // don't recurse into repos
		}

		// Recurse
		sub, err := scanDirRecursive(entryPath, ignore, depth+1)
		if err != nil {
			return nil, err
		}
		repos = append(repos, sub...)
	}

	return repos, nil
}
