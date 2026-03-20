package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

type GitHubCollector struct{}

func (g *GitHubCollector) Name() string      { return "github" }
func (g *GitHubCollector) EnvVars() []string { return []string{"GITHUB_TOKEN"} }

func (g *GitHubCollector) Enabled(cfg *config.Config) bool {
	if !cfg.AdapterEnabled("github") {
		return false
	}
	return os.Getenv("GITHUB_TOKEN") != ""
}

func (g *GitHubCollector) Collect(ctx context.Context, s store.Store, cfg *config.Config, syncID int64) error {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN not set")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/notifications", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching notifications: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var rawNotifs []json.RawMessage
	if err := json.Unmarshal(body, &rawNotifs); err != nil {
		return fmt.Errorf("parsing notifications: %w", err)
	}

	var notifs []domain.Notification
	for _, raw := range rawNotifs {
		n, err := ParseGitHubNotification(raw)
		if err != nil {
			continue
		}
		notifs = append(notifs, n)
	}

	return s.SaveGitHubNotifications(ctx, syncID, notifs)
}

type ghNotification struct {
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Subject struct {
		Title string `json:"title"`
		Type  string `json:"type"`
		URL   string `json:"url"`
	} `json:"subject"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ParseGitHubNotification parses a single GitHub notification JSON.
func ParseGitHubNotification(data []byte) (domain.Notification, error) {
	var gh ghNotification
	if err := json.Unmarshal(data, &gh); err != nil {
		return domain.Notification{}, err
	}

	notifType := mapGitHubType(gh.Subject.Type)

	// Convert API URL to web URL
	webURL := gh.Subject.URL
	webURL = strings.Replace(webURL, "api.github.com/repos/", "github.com/", 1)
	webURL = strings.Replace(webURL, "/pulls/", "/pull/", 1)

	return domain.Notification{
		RepoName:  gh.Repository.FullName,
		Type:      notifType,
		Title:     gh.Subject.Title,
		URL:       webURL,
		State:     "unread",
		UpdatedAt: gh.UpdatedAt,
	}, nil
}

func mapGitHubType(ghType string) string {
	switch ghType {
	case "PullRequest":
		return "pr"
	case "Issue":
		return "issue"
	case "CheckSuite", "CheckRun":
		return "ci"
	default:
		return strings.ToLower(ghType)
	}
}

func init() {
	Register(&GitHubCollector{})
}
