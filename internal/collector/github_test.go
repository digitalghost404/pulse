package collector_test

import (
	"testing"

	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
)

func TestGitHubCollector_ParseNotification(t *testing.T) {
	tests := []struct {
		name string
		json string
		want domain.Notification
	}{
		{
			name: "pull_request",
			json: `{"repository":{"full_name":"xcoleman/obsidian-mcp"},"subject":{"title":"Fix FTS5 indexing","type":"PullRequest","url":"https://api.github.com/repos/xcoleman/obsidian-mcp/pulls/42"},"updated_at":"2026-03-20T10:00:00Z"}`,
			want: domain.Notification{
				RepoName: "xcoleman/obsidian-mcp",
				Type:     "pr",
				Title:    "Fix FTS5 indexing",
			},
		},
		{
			name: "issue",
			json: `{"repository":{"full_name":"xcoleman/cortex"},"subject":{"title":"Bug in RAG chunking","type":"Issue","url":"https://api.github.com/repos/xcoleman/cortex/issues/10"},"updated_at":"2026-03-20T08:00:00Z"}`,
			want: domain.Notification{
				RepoName: "xcoleman/cortex",
				Type:     "issue",
				Title:    "Bug in RAG chunking",
			},
		},
		{
			name: "ci_check",
			json: `{"repository":{"full_name":"xcoleman/cortex"},"subject":{"title":"CI build failed","type":"CheckSuite","url":"https://api.github.com/repos/xcoleman/cortex/check-suites/1"},"updated_at":"2026-03-20T09:00:00Z"}`,
			want: domain.Notification{
				RepoName: "xcoleman/cortex",
				Type:     "ci",
				Title:    "CI build failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := collector.ParseGitHubNotification([]byte(tt.json))
			if err != nil {
				t.Fatalf("ParseGitHubNotification: %v", err)
			}
			if got.RepoName != tt.want.RepoName {
				t.Errorf("repo: got %s, want %s", got.RepoName, tt.want.RepoName)
			}
			if got.Type != tt.want.Type {
				t.Errorf("type: got %s, want %s", got.Type, tt.want.Type)
			}
			if got.Title != tt.want.Title {
				t.Errorf("title: got %s, want %s", got.Title, tt.want.Title)
			}
		})
	}
}

func TestGitHubCollector_EnabledCheck(t *testing.T) {
	gc := &collector.GitHubCollector{}

	// Set token so Enabled returns true when adapter config allows it
	t.Setenv("GITHUB_TOKEN", "test-token")

	cfg := &config.Config{}
	if !gc.Enabled(cfg) {
		t.Error("expected github collector enabled by default")
	}

	cfg = &config.Config{Adapters: map[string]bool{"github": false}}
	if gc.Enabled(cfg) {
		t.Error("expected github collector disabled")
	}
}

func TestGitHubCollector_EnvVars(t *testing.T) {
	gc := &collector.GitHubCollector{}
	envVars := gc.EnvVars()
	if len(envVars) != 1 || envVars[0] != "GITHUB_TOKEN" {
		t.Errorf("expected [GITHUB_TOKEN], got %v", envVars)
	}
}
