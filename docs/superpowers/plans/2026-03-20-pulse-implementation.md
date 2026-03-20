# Pulse Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a personal command center CLI that combines morning briefing, project health monitoring, and AI cost tracking into a single Go binary.

**Architecture:** Single Go binary with Cobra CLI, Bubble Tea TUI, and SQLite storage. Cron-based data collection via pluggable Collector interface; instant reads via Briefing Engine dispatched to pluggable Writers (stdout, TUI, Obsidian).

**Tech Stack:** Go 1.22+, Cobra, Bubble Tea, modernc/sqlite, sqlc, Viper

**Spec:** `docs/superpowers/specs/2026-03-20-pulse-design.md`

---

## File Structure

```
pulse/
├── cmd/
│   └── pulse/
│       └── main.go                    # Entry point — initializes and executes root command
├── internal/
│   ├── config/
│   │   ├── config.go                  # Config struct, Load(), default generation
│   │   └── config_test.go
│   ├── domain/
│   │   └── types.go                   # Shared domain types (GitSnapshot, CostEntry, Briefing, etc.)
│   ├── store/
│   │   ├── store.go                   # Store interface definition
│   │   ├── sqlite.go                  # SQLite Store implementation
│   │   ├── sqlite_test.go            # Integration tests (real DB)
│   │   └── migrations/
│   │       └── 001_initial.sql        # Initial schema
│   ├── discovery/
│   │   ├── discovery.go               # Project discovery (scan dirs, find .git repos)
│   │   └── discovery_test.go
│   ├── collector/
│   │   ├── registry.go                # Collector interface + global registry
│   │   ├── git.go                     # Git scanner collector
│   │   ├── git_test.go
│   │   ├── github.go                  # GitHub notifications collector
│   │   ├── github_test.go
│   │   ├── system.go                  # System resources collector
│   │   ├── system_test.go
│   │   ├── docker.go                  # Docker status collector
│   │   ├── docker_test.go
│   │   ├── cost_stub.go               # Stub cost collectors (Claude, Voyage, Tavily, ElevenLabs)
│   │   └── cost_stub_test.go
│   ├── sync/
│   │   ├── engine.go                  # Sync engine — orchestrates collectors
│   │   └── engine_test.go
│   ├── briefing/
│   │   ├── engine.go                  # Briefing engine — reads DB, composes Briefing
│   │   └── engine_test.go
│   ├── writer/
│   │   ├── registry.go                # Writer interface + registry
│   │   ├── stdout.go                  # Stdout writer (colored text)
│   │   ├── stdout_test.go
│   │   ├── obsidian.go                # Obsidian daily note writer
│   │   ├── obsidian_test.go
│   │   └── testdata/                  # Golden files for stdout output
│   │       └── briefing_full.golden
│   ├── tui/
│   │   ├── app.go                     # Root Bubble Tea model, tab switching
│   │   ├── briefing_tab.go            # Briefing tab view
│   │   ├── projects_tab.go            # Projects tab with drill-down
│   │   ├── costs_tab.go               # Costs tab with drill-down
│   │   ├── styles.go                  # Shared lipgloss styles
│   │   └── app_test.go                # teatest-based TUI tests
│   └── cli/
│       ├── root.go                    # Root command (prints briefing)
│       ├── sync_cmd.go                # pulse sync [--only]
│       ├── tui_cmd.go                 # pulse tui
│       ├── obsidian_cmd.go            # pulse obsidian
│       ├── costs_cmd.go               # pulse costs [--service] [--period]
│       ├── projects_cmd.go            # pulse projects [--repo]
│       ├── config_cmd.go              # pulse config {init,show,adapters}
│       └── version_cmd.go             # pulse version
├── go.mod
└── go.sum
```

---

## Task 1: Project Scaffold & Go Module

**Files:**
- Create: `go.mod`
- Create: `cmd/pulse/main.go`
- Create: `internal/cli/root.go`
- Create: `internal/cli/version_cmd.go`

- [ ] **Step 1: Initialize Go module**

Run: `go mod init github.com/xcoleman/pulse`

- [ ] **Step 2: Install core dependencies**

Run:
```bash
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get modernc.org/sqlite@latest
```

- [ ] **Step 3: Create root command**

`internal/cli/root.go`:
```go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "pulse",
	Short: "Personal command center — briefing, project health, cost tracking",
	Long:  "Pulse synthesizes signals from your projects, AI services, and dev environment into a single morning briefing.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("pulse: no data yet — run 'pulse sync' first")
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug output")
	rootCmd.PersistentFlags().Bool("json", false, "Output as JSON")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Create version command**

`internal/cli/version_cmd.go`:
```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Pulse version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("pulse %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
```

- [ ] **Step 5: Create main.go entry point**

`cmd/pulse/main.go`:
```go
package main

import "github.com/xcoleman/pulse/internal/cli"

func main() {
	cli.Execute()
}
```

- [ ] **Step 6: Build and verify**

Run: `go build -o pulse ./cmd/pulse && ./pulse version`
Expected: `pulse dev`

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat: scaffold project with Go module, root command, and version command"
```

---

## Task 2: Config

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write config tests**

`internal/config/config_test.go`:
```go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xcoleman/pulse/internal/config"
)

func TestLoadConfig_DefaultValues(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// No config file — should return defaults
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Sync.Timeout != "30s" {
		t.Errorf("expected default timeout 30s, got %s", cfg.Sync.Timeout)
	}
	if cfg.Costs.DefaultPeriod != "30d" {
		t.Errorf("expected default period 30d, got %s", cfg.Costs.DefaultPeriod)
	}
	if cfg.Costs.Currency != "USD" {
		t.Errorf("expected default currency USD, got %s", cfg.Costs.Currency)
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `projects:
  scan:
    - /tmp/repos
  ignore:
    - vendor
github:
  username: testuser
adapters:
  git: true
  github: false
sync:
  timeout: 60s
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Projects.Scan) != 1 || cfg.Projects.Scan[0] != "/tmp/repos" {
		t.Errorf("expected scan dirs [/tmp/repos], got %v", cfg.Projects.Scan)
	}
	if cfg.GitHub.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", cfg.GitHub.Username)
	}
	if cfg.Sync.Timeout != "60s" {
		t.Errorf("expected timeout 60s, got %s", cfg.Sync.Timeout)
	}
	if cfg.Adapters["github"] != false {
		t.Errorf("expected github adapter disabled")
	}
}

func TestLoadConfig_AdapterEnabled(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `adapters:
  git: true
  github: false
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.AdapterEnabled("git") {
		t.Error("expected git adapter enabled")
	}
	if cfg.AdapterEnabled("github") {
		t.Error("expected github adapter disabled")
	}
	// Unlisted adapters default to enabled
	if !cfg.AdapterEnabled("docker") {
		t.Error("expected unlisted adapter to default to enabled")
	}
}

func TestGenerateDefault(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	err := config.GenerateDefault(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatal("config file not created")
	}

	// Verify it loads correctly
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("generated config not loadable: %v", err)
	}

	if cfg.Sync.Timeout != "30s" {
		t.Errorf("expected default timeout in generated config")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/...`
Expected: Compilation error — package doesn't exist yet

- [ ] **Step 3: Implement config package**

`internal/config/config.go`:
```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Projects ProjectsConfig         `mapstructure:"projects"`
	GitHub   GitHubConfig           `mapstructure:"github"`
	Obsidian ObsidianConfig         `mapstructure:"obsidian"`
	Adapters map[string]bool        `mapstructure:"adapters"`
	Sync     SyncConfig             `mapstructure:"sync"`
	Costs    CostsConfig            `mapstructure:"costs"`
}

type ProjectsConfig struct {
	Scan   []string `mapstructure:"scan"`
	Ignore []string `mapstructure:"ignore"`
}

type GitHubConfig struct {
	Username string `mapstructure:"username"`
}

type ObsidianConfig struct {
	VaultPath      string `mapstructure:"vault_path"`
	DailyNotePath  string `mapstructure:"daily_note_path"`
	SectionHeading string `mapstructure:"section_heading"`
}

type SyncConfig struct {
	Timeout string `mapstructure:"timeout"`
	LogFile string `mapstructure:"log_file"`
}

type CostsConfig struct {
	DefaultPeriod string `mapstructure:"default_period"`
	Currency      string `mapstructure:"currency"`
}

// AdapterEnabled returns whether an adapter is enabled. Unlisted adapters default to enabled.
func (c *Config) AdapterEnabled(name string) bool {
	if enabled, ok := c.Adapters[name]; ok {
		return enabled
	}
	return true
}

// DefaultConfigDir returns ~/.config/pulse/
func DefaultConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "pulse")
}

// DefaultConfigPath returns ~/.config/pulse/config.yaml
func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir(), "config.yaml")
}

// ObsidianDailyNotePath resolves the daily note path with date tokens.
// Translates Obsidian-style tokens (YYYY, MM, DD) to Go time format.
func (c *Config) ObsidianDailyNotePath(t interface{ Format(string) string }) string {
	path := c.Obsidian.DailyNotePath
	// Replace Obsidian tokens with Go time format placeholders, then format
	path = strings.ReplaceAll(path, "YYYY", "2006")
	path = strings.ReplaceAll(path, "MM", "01")
	path = strings.ReplaceAll(path, "DD", "02")
	return filepath.Join(c.Obsidian.VaultPath, t.Format(path))
}

func Load(path string) (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("sync.timeout", "30s")
	v.SetDefault("costs.default_period", "30d")
	v.SetDefault("costs.currency", "USD")
	v.SetDefault("obsidian.section_heading", "## Pulse Briefing")

	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				if !os.IsNotExist(err) {
					return nil, fmt.Errorf("reading config: %w", err)
				}
			}
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

const defaultConfigTemplate = `# Pulse configuration
projects:
  scan:
    - ~/projects-wsl
  ignore:
    - voidterm-builds
    - docs

github:
  username: ""

obsidian:
  vault_path: ""
  daily_note_path: "Daily Notes/YYYY-MM-DD.md"
  section_heading: "## Pulse Briefing"

adapters:
  git: true
  github: true
  claude: true
  voyage: true
  tavily: true
  elevenlabs: true
  ollama: false
  docker: true
  system: true

sync:
  timeout: 30s
  # log_file: ~/.config/pulse/sync.log

costs:
  default_period: 30d
  currency: USD
`

func GenerateDefault(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	return os.WriteFile(path, []byte(defaultConfigTemplate), 0644)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/... -v`
Expected: All 4 tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package with YAML loading, defaults, and generation"
```

---

## Task 3: Domain Types

**Files:**
- Create: `internal/domain/types.go`

- [ ] **Step 1: Create domain types**

`internal/domain/types.go`:
```go
package domain

import "time"

// GitSnapshot represents the state of a git repo at a point in time.
type GitSnapshot struct {
	RepoPath       string
	RepoName       string
	Branch         string
	DirtyFiles     int
	Ahead          int
	Behind         int
	LastCommitHash string
	LastCommitMsg  string
	LastCommitAt   time.Time
}

// GitBranch represents a branch in a git repo.
type GitBranch struct {
	RepoPath     string
	BranchName   string
	LastCommitAt time.Time
	IsMerged     bool
	IsCurrent    bool
}

// Notification represents a GitHub notification.
type Notification struct {
	RepoName  string
	Type      string // pr, issue, ci
	Title     string
	URL       string
	State     string
	UpdatedAt time.Time
}

// CostEntry represents a normalized cost record from any service.
type CostEntry struct {
	Service       string
	PeriodStart   time.Time
	PeriodEnd     time.Time
	AmountCents   int
	Currency      string
	UsageQuantity float64
	UsageUnit     string
	RawData       string // JSON
}

// DockerSnapshot represents the state of a Docker container.
type DockerSnapshot struct {
	ContainerName string
	Image         string
	Status        string
	Ports         string // JSON
	CPUPct        float64
	MemoryMB      float64
}

// SystemSnapshot represents system resource usage.
type SystemSnapshot struct {
	CPUPct        float64
	MemoryUsedMB  float64
	MemoryTotalMB float64
	DiskUsedGB    float64
	DiskTotalGB   float64
}

// SyncRun represents a single sync execution.
type SyncRun struct {
	ID          int64
	StartedAt   time.Time
	CompletedAt time.Time
	Status      string // success, partial, failed
	Error       string
}

// BriefingEntry represents a rendered briefing stored in history.
type BriefingEntry struct {
	ID        int64
	CreatedAt time.Time
	Content   string
	Writer    string
}

// Briefing is the intermediate representation between the DB and Writers.
type Briefing struct {
	GeneratedAt   time.Time
	Projects      []ProjectSummary
	Notifications []Notification
	CostSummary   CostSummary
	Docker        []DockerSnapshot
	System        SystemSnapshot
}

// ProjectSummary combines git snapshot with branch info for display.
type ProjectSummary struct {
	GitSnapshot
	Branches []GitBranch
}

// CostSummary aggregates cost data for the briefing.
type CostSummary struct {
	TotalCents   int
	Currency     string
	ByService    []ServiceCost
	Period       string
	BurnRateCents int // daily average
}

// ServiceCost represents cost for a single service.
type ServiceCost struct {
	Service       string
	AmountCents   int
	UsageQuantity float64
	UsageUnit     string
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/domain/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/domain/
git commit -m "feat: add domain types for all data models"
```

---

## Task 4: Store Interface & SQLite Implementation

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/sqlite.go`
- Create: `internal/store/sqlite_test.go`
- Create: `internal/store/migrations/001_initial.sql`

- [ ] **Step 1: Create the initial migration SQL**

`internal/store/migrations/001_initial.sql`:
```sql
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER NOT NULL
);
INSERT INTO schema_version (version) VALUES (1);

CREATE TABLE IF NOT EXISTS sync_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    status TEXT NOT NULL DEFAULT 'running',
    error TEXT
);

CREATE TABLE IF NOT EXISTS git_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id INTEGER NOT NULL REFERENCES sync_runs(id),
    repo_path TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    branch TEXT NOT NULL,
    dirty_files INTEGER NOT NULL DEFAULT 0,
    ahead INTEGER NOT NULL DEFAULT 0,
    behind INTEGER NOT NULL DEFAULT 0,
    last_commit_hash TEXT,
    last_commit_msg TEXT,
    last_commit_at DATETIME
);

CREATE TABLE IF NOT EXISTS git_branches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id INTEGER NOT NULL REFERENCES sync_runs(id),
    repo_path TEXT NOT NULL,
    branch_name TEXT NOT NULL,
    last_commit_at DATETIME,
    is_merged BOOLEAN NOT NULL DEFAULT 0,
    is_current BOOLEAN NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS github_notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id INTEGER NOT NULL REFERENCES sync_runs(id),
    repo_name TEXT NOT NULL,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    url TEXT,
    state TEXT,
    updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS cost_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id INTEGER NOT NULL REFERENCES sync_runs(id),
    service TEXT NOT NULL,
    period_start DATETIME,
    period_end DATETIME,
    amount_cents INTEGER NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'USD',
    usage_quantity REAL,
    usage_unit TEXT,
    raw_data TEXT
);

CREATE TABLE IF NOT EXISTS docker_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id INTEGER NOT NULL REFERENCES sync_runs(id),
    container_name TEXT NOT NULL,
    image TEXT,
    status TEXT,
    ports TEXT,
    cpu_pct REAL,
    memory_mb REAL
);

CREATE TABLE IF NOT EXISTS system_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id INTEGER NOT NULL REFERENCES sync_runs(id),
    cpu_pct REAL,
    memory_used_mb REAL,
    memory_total_mb REAL,
    disk_used_gb REAL,
    disk_total_gb REAL
);

CREATE TABLE IF NOT EXISTS briefing_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL,
    content TEXT NOT NULL,
    writer TEXT NOT NULL
);

```

Note: `PRAGMA journal_mode=WAL` is set in `NewSQLite()` on connection open, not in migrations.

- [ ] **Step 2: Create Store interface**

`internal/store/store.go`:
```go
package store

import (
	"context"
	"time"

	"github.com/xcoleman/pulse/internal/domain"
)

// Store defines the data access interface. Concrete implementation uses SQLite.
// Tests can mock this interface.
type Store interface {
	// Sync run management
	CreateSyncRun(ctx context.Context) (int64, error)
	CompleteSyncRun(ctx context.Context, id int64, status string, syncErr string) error

	// Write methods (used by collectors)
	SaveGitSnapshot(ctx context.Context, syncID int64, snapshot domain.GitSnapshot) error
	SaveGitBranches(ctx context.Context, syncID int64, branches []domain.GitBranch) error
	SaveCostEntry(ctx context.Context, syncID int64, entry domain.CostEntry) error
	SaveDockerSnapshot(ctx context.Context, syncID int64, snapshot domain.DockerSnapshot) error
	SaveSystemSnapshot(ctx context.Context, syncID int64, snapshot domain.SystemSnapshot) error
	SaveGitHubNotifications(ctx context.Context, syncID int64, notifs []domain.Notification) error
	SaveBriefing(ctx context.Context, entry domain.BriefingEntry) error

	// Read methods (used by briefing engine)
	LatestSyncID(ctx context.Context) (int64, error)
	GetGitSnapshots(ctx context.Context, syncID int64) ([]domain.GitSnapshot, error)
	GetGitBranches(ctx context.Context, syncID int64, repoPath string) ([]domain.GitBranch, error)
	GetGitHubNotifications(ctx context.Context, syncID int64) ([]domain.Notification, error)
	GetCostEntries(ctx context.Context, since time.Time) ([]domain.CostEntry, error)
	GetDockerSnapshots(ctx context.Context, syncID int64) ([]domain.DockerSnapshot, error)
	GetSystemSnapshot(ctx context.Context, syncID int64) (*domain.SystemSnapshot, error)
	GetLastBriefingTime(ctx context.Context) (time.Time, error)

	// Maintenance
	PruneBriefingHistory(ctx context.Context, olderThan time.Time) (int64, error)

	// Lifecycle
	Close() error
}
```

- [ ] **Step 3: Write Store integration tests**

`internal/store/sqlite_test.go`:
```go
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
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `go test ./internal/store/... -v`
Expected: Compilation error — `store.NewSQLite` doesn't exist

- [ ] **Step 5: Implement SQLite store**

`internal/store/sqlite.go`:
```go
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/xcoleman/pulse/internal/domain"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLite(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for concurrent read/write safety
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return s, nil
}

func (s *SQLiteStore) migrate() error {
	// Ensure schema_version table exists
	s.db.Exec("CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)")

	var version int
	err := s.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version)
	if err != nil {
		version = 0
	}

	// Read all migration files and run any with version > current
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	for _, entry := range entries {
		// Parse version from filename: "001_initial.sql" → 1
		name := entry.Name()
		var migVersion int
		fmt.Sscanf(name, "%d_", &migVersion)
		if migVersion <= version {
			continue
		}

		data, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", name, err)
		}
		if _, err := s.db.Exec(string(data)); err != nil {
			return fmt.Errorf("executing migration %s: %w", name, err)
		}
	}

	return nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// --- Sync Runs ---

func (s *SQLiteStore) CreateSyncRun(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO sync_runs (started_at, status) VALUES (?, 'running')",
		time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) CompleteSyncRun(ctx context.Context, id int64, status string, syncErr string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE sync_runs SET completed_at = ?, status = ?, error = ? WHERE id = ?",
		time.Now().UTC(), status, syncErr, id)
	return err
}

func (s *SQLiteStore) LatestSyncID(ctx context.Context) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx,
		"SELECT id FROM sync_runs WHERE status IN ('success', 'partial') ORDER BY id DESC LIMIT 1").Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

// --- Git ---

func (s *SQLiteStore) SaveGitSnapshot(ctx context.Context, syncID int64, snap domain.GitSnapshot) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO git_snapshots (sync_id, repo_path, repo_name, branch, dirty_files, ahead, behind, last_commit_hash, last_commit_msg, last_commit_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		syncID, snap.RepoPath, snap.RepoName, snap.Branch, snap.DirtyFiles,
		snap.Ahead, snap.Behind, snap.LastCommitHash, snap.LastCommitMsg, snap.LastCommitAt.UTC())
	return err
}

func (s *SQLiteStore) SaveGitBranches(ctx context.Context, syncID int64, branches []domain.GitBranch) error {
	for _, b := range branches {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO git_branches (sync_id, repo_path, branch_name, last_commit_at, is_merged, is_current)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			syncID, b.RepoPath, b.BranchName, b.LastCommitAt.UTC(), b.IsMerged, b.IsCurrent)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) GetGitSnapshots(ctx context.Context, syncID int64) ([]domain.GitSnapshot, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT repo_path, repo_name, branch, dirty_files, ahead, behind, last_commit_hash, last_commit_msg, last_commit_at
		 FROM git_snapshots WHERE sync_id = ?`, syncID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.GitSnapshot
	for rows.Next() {
		var snap domain.GitSnapshot
		if err := rows.Scan(&snap.RepoPath, &snap.RepoName, &snap.Branch, &snap.DirtyFiles,
			&snap.Ahead, &snap.Behind, &snap.LastCommitHash, &snap.LastCommitMsg, &snap.LastCommitAt); err != nil {
			return nil, err
		}
		result = append(result, snap)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) GetGitBranches(ctx context.Context, syncID int64, repoPath string) ([]domain.GitBranch, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT repo_path, branch_name, last_commit_at, is_merged, is_current
		 FROM git_branches WHERE sync_id = ? AND repo_path = ?`, syncID, repoPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.GitBranch
	for rows.Next() {
		var b domain.GitBranch
		if err := rows.Scan(&b.RepoPath, &b.BranchName, &b.LastCommitAt, &b.IsMerged, &b.IsCurrent); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

// --- GitHub ---

func (s *SQLiteStore) SaveGitHubNotifications(ctx context.Context, syncID int64, notifs []domain.Notification) error {
	for _, n := range notifs {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO github_notifications (sync_id, repo_name, type, title, url, state, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			syncID, n.RepoName, n.Type, n.Title, n.URL, n.State, n.UpdatedAt.UTC())
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) GetGitHubNotifications(ctx context.Context, syncID int64) ([]domain.Notification, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT repo_name, type, title, url, state, updated_at
		 FROM github_notifications WHERE sync_id = ?`, syncID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Notification
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.RepoName, &n.Type, &n.Title, &n.URL, &n.State, &n.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, rows.Err()
}

// --- Costs ---

func (s *SQLiteStore) SaveCostEntry(ctx context.Context, syncID int64, entry domain.CostEntry) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cost_entries (sync_id, service, period_start, period_end, amount_cents, currency, usage_quantity, usage_unit, raw_data)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		syncID, entry.Service, entry.PeriodStart.UTC(), entry.PeriodEnd.UTC(),
		entry.AmountCents, entry.Currency, entry.UsageQuantity, entry.UsageUnit, entry.RawData)
	return err
}

func (s *SQLiteStore) GetCostEntries(ctx context.Context, since time.Time) ([]domain.CostEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT service, period_start, period_end, amount_cents, currency, usage_quantity, usage_unit, raw_data
		 FROM cost_entries WHERE period_end >= ? ORDER BY period_end DESC`, since.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.CostEntry
	for rows.Next() {
		var e domain.CostEntry
		if err := rows.Scan(&e.Service, &e.PeriodStart, &e.PeriodEnd, &e.AmountCents,
			&e.Currency, &e.UsageQuantity, &e.UsageUnit, &e.RawData); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// --- Docker ---

func (s *SQLiteStore) SaveDockerSnapshot(ctx context.Context, syncID int64, snap domain.DockerSnapshot) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO docker_snapshots (sync_id, container_name, image, status, ports, cpu_pct, memory_mb)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		syncID, snap.ContainerName, snap.Image, snap.Status, snap.Ports, snap.CPUPct, snap.MemoryMB)
	return err
}

func (s *SQLiteStore) GetDockerSnapshots(ctx context.Context, syncID int64) ([]domain.DockerSnapshot, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT container_name, image, status, ports, cpu_pct, memory_mb
		 FROM docker_snapshots WHERE sync_id = ?`, syncID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.DockerSnapshot
	for rows.Next() {
		var snap domain.DockerSnapshot
		if err := rows.Scan(&snap.ContainerName, &snap.Image, &snap.Status, &snap.Ports, &snap.CPUPct, &snap.MemoryMB); err != nil {
			return nil, err
		}
		result = append(result, snap)
	}
	return result, rows.Err()
}

// --- System ---

func (s *SQLiteStore) SaveSystemSnapshot(ctx context.Context, syncID int64, snap domain.SystemSnapshot) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO system_snapshots (sync_id, cpu_pct, memory_used_mb, memory_total_mb, disk_used_gb, disk_total_gb)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		syncID, snap.CPUPct, snap.MemoryUsedMB, snap.MemoryTotalMB, snap.DiskUsedGB, snap.DiskTotalGB)
	return err
}

func (s *SQLiteStore) GetSystemSnapshot(ctx context.Context, syncID int64) (*domain.SystemSnapshot, error) {
	var snap domain.SystemSnapshot
	err := s.db.QueryRowContext(ctx,
		`SELECT cpu_pct, memory_used_mb, memory_total_mb, disk_used_gb, disk_total_gb
		 FROM system_snapshots WHERE sync_id = ?`, syncID).
		Scan(&snap.CPUPct, &snap.MemoryUsedMB, &snap.MemoryTotalMB, &snap.DiskUsedGB, &snap.DiskTotalGB)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

// --- Briefing History ---

func (s *SQLiteStore) SaveBriefing(ctx context.Context, entry domain.BriefingEntry) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO briefing_history (created_at, content, writer) VALUES (?, ?, ?)`,
		entry.CreatedAt.UTC(), entry.Content, entry.Writer)
	return err
}

func (s *SQLiteStore) GetLastBriefingTime(ctx context.Context) (time.Time, error) {
	var t time.Time
	err := s.db.QueryRowContext(ctx,
		"SELECT created_at FROM briefing_history ORDER BY created_at DESC LIMIT 1").Scan(&t)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	return t, err
}

func (s *SQLiteStore) PruneBriefingHistory(ctx context.Context, olderThan time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		"DELETE FROM briefing_history WHERE created_at < ?", olderThan.UTC())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/store/... -v`
Expected: All 8 tests pass

- [ ] **Step 7: Commit**

```bash
git add internal/store/ internal/domain/
git commit -m "feat: add Store interface, SQLite implementation, and initial migration"
```

---

## Task 5: Project Discovery

**Files:**
- Create: `internal/discovery/discovery.go`
- Create: `internal/discovery/discovery_test.go`

- [ ] **Step 1: Write discovery tests**

`internal/discovery/discovery_test.go`:
```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/discovery/... -v`
Expected: Compilation error

- [ ] **Step 3: Implement discovery**

`internal/discovery/discovery.go`:
```go
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
	if depth > maxDepth {
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/discovery/... -v`
Expected: All 5 tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/
git commit -m "feat: add project discovery with depth limiting and exclusions"
```

---

## Task 6: Collector Interface & Registry

**Files:**
- Create: `internal/collector/registry.go`

- [ ] **Step 1: Create collector interface and registry**

`internal/collector/registry.go`:
```go
package collector

import (
	"context"
	"fmt"
	"sync"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
)

// Collector gathers data from an external source and writes it to the store.
type Collector interface {
	Name() string
	EnvVars() []string
	Enabled(cfg *config.Config) bool
	Collect(ctx context.Context, s store.Store, cfg *config.Config, syncID int64) error
}

var (
	mu       sync.RWMutex
	registry = make(map[string]Collector)
)

// Register adds a collector to the global registry.
func Register(c Collector) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[c.Name()]; exists {
		panic(fmt.Sprintf("collector %q already registered", c.Name()))
	}
	registry[c.Name()] = c
}

// Get returns a collector by name.
func Get(name string) (Collector, bool) {
	mu.RLock()
	defer mu.RUnlock()
	c, ok := registry[name]
	return c, ok
}

// All returns all registered collectors.
func All() []Collector {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Collector, 0, len(registry))
	for _, c := range registry {
		result = append(result, c)
	}
	return result
}

// Enabled returns all collectors that are enabled in the config.
func Enabled(cfg *config.Config) []Collector {
	all := All()
	result := make([]Collector, 0, len(all))
	for _, c := range all {
		if c.Enabled(cfg) {
			result = append(result, c)
		}
	}
	return result
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/collector/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/collector/registry.go
git commit -m "feat: add Collector interface and global registry"
```

---

## Task 7: Git Collector

**Files:**
- Create: `internal/collector/git.go`
- Create: `internal/collector/git_test.go`

- [ ] **Step 1: Write git collector tests**

`internal/collector/git_test.go`:
```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/collector/... -v`
Expected: Compilation error

- [ ] **Step 3: Implement git collector**

`internal/collector/git.go`:
```go
package collector

import (
	"context"
	"os/exec"
	"path/filepath"
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/collector/... -v`
Expected: All 3 tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/collector/git.go internal/collector/git_test.go
git commit -m "feat: add git collector with repo scanning and branch tracking"
```

---

## Task 8: System Resources Collector

**Files:**
- Create: `internal/collector/system.go`
- Create: `internal/collector/system_test.go`

- [ ] **Step 1: Write system collector tests**

`internal/collector/system_test.go`:
```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/collector/... -run TestSystem -v`
Expected: Compilation error

- [ ] **Step 3: Implement system collector**

`internal/collector/system.go`:
```go
package collector

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

type SystemCollector struct{}

func (s *SystemCollector) Name() string      { return "system" }
func (s *SystemCollector) EnvVars() []string { return nil }

func (s *SystemCollector) Enabled(cfg *config.Config) bool {
	return cfg.AdapterEnabled("system")
}

func (s *SystemCollector) Collect(ctx context.Context, st store.Store, cfg *config.Config, syncID int64) error {
	snap := domain.SystemSnapshot{}

	// CPU from /proc/stat (snapshot — not a delta, but good enough for a point-in-time view)
	snap.CPUPct = readCPUPercent()

	// Memory from /proc/meminfo
	snap.MemoryTotalMB, snap.MemoryUsedMB = readMemory()

	// Disk from syscall.Statfs
	snap.DiskTotalGB, snap.DiskUsedGB = readDisk("/")

	return st.SaveSystemSnapshot(ctx, syncID, snap)
}

func readCPUPercent() float64 {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				user, _ := strconv.ParseFloat(fields[1], 64)
				nice, _ := strconv.ParseFloat(fields[2], 64)
				system, _ := strconv.ParseFloat(fields[3], 64)
				idle, _ := strconv.ParseFloat(fields[4], 64)
				total := user + nice + system + idle
				if total > 0 {
					return (total - idle) / total * 100
				}
			}
		}
	}
	return 0
}

func readMemory() (totalMB, usedMB float64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	var totalKB, availKB float64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				totalKB, _ = strconv.ParseFloat(fields[1], 64)
			}
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				availKB, _ = strconv.ParseFloat(fields[1], 64)
			}
		}
	}
	totalMB = totalKB / 1024
	usedMB = (totalKB - availKB) / 1024
	return
}

func readDisk(path string) (totalGB, usedGB float64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0
	}
	totalBytes := float64(stat.Blocks) * float64(stat.Bsize)
	freeBytes := float64(stat.Bfree) * float64(stat.Bsize)
	totalGB = totalBytes / (1024 * 1024 * 1024)
	usedGB = (totalBytes - freeBytes) / (1024 * 1024 * 1024)
	return
}

func init() {
	Register(&SystemCollector{})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/collector/... -run TestSystem -v`
Expected: Both tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/collector/system.go internal/collector/system_test.go
git commit -m "feat: add system resources collector (CPU, memory, disk from /proc)"
```

---

## Task 9: Docker Collector

**Files:**
- Create: `internal/collector/docker.go`
- Create: `internal/collector/docker_test.go`

- [ ] **Step 1: Write docker collector tests**

`internal/collector/docker_test.go`:
```go
package collector_test

import (
	"testing"

	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
)

func TestDockerCollector_ParseDockerPS(t *testing.T) {
	// Table-driven: mock docker ps JSON output → expected snapshots
	tests := []struct {
		name     string
		jsonLine string
		want     domain.DockerSnapshot
	}{
		{
			name:     "running_container",
			jsonLine: `{"Names":"redis","Image":"redis:7","Status":"Up 2 hours","Ports":"0.0.0.0:6379->6379/tcp"}`,
			want: domain.DockerSnapshot{
				ContainerName: "redis",
				Image:         "redis:7",
				Status:        "Up 2 hours",
				Ports:         "0.0.0.0:6379->6379/tcp",
			},
		},
		{
			name:     "stopped_container",
			jsonLine: `{"Names":"postgres","Image":"postgres:16","Status":"Exited (0) 3 hours ago","Ports":""}`,
			want: domain.DockerSnapshot{
				ContainerName: "postgres",
				Image:         "postgres:16",
				Status:        "Exited (0) 3 hours ago",
				Ports:         "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := collector.ParseDockerPSLine(tt.jsonLine)
			if err != nil {
				t.Fatalf("ParseDockerPSLine: %v", err)
			}
			if got.ContainerName != tt.want.ContainerName {
				t.Errorf("name: got %s, want %s", got.ContainerName, tt.want.ContainerName)
			}
			if got.Image != tt.want.Image {
				t.Errorf("image: got %s, want %s", got.Image, tt.want.Image)
			}
			if got.Status != tt.want.Status {
				t.Errorf("status: got %s, want %s", got.Status, tt.want.Status)
			}
		})
	}
}

func TestDockerCollector_EnabledCheck(t *testing.T) {
	dc := &collector.DockerCollector{}

	cfg := &config.Config{}
	if !dc.Enabled(cfg) {
		t.Error("expected docker collector enabled by default")
	}

	cfg = &config.Config{Adapters: map[string]bool{"docker": false}}
	if dc.Enabled(cfg) {
		t.Error("expected docker collector disabled")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/collector/... -run TestDocker -v`
Expected: Compilation error

- [ ] **Step 3: Implement docker collector**

`internal/collector/docker.go`:
```go
package collector

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

type DockerCollector struct{}

func (d *DockerCollector) Name() string      { return "docker" }
func (d *DockerCollector) EnvVars() []string { return nil }

func (d *DockerCollector) Enabled(cfg *config.Config) bool {
	return cfg.AdapterEnabled("docker")
}

func (d *DockerCollector) Collect(ctx context.Context, s store.Store, cfg *config.Config, syncID int64) error {
	// Check if docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		return nil // docker not installed, skip silently
	}

	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", `{"Names":"{{.Names}}","Image":"{{.Image}}","Status":"{{.Status}}","Ports":"{{.Ports}}"}`)
	out, err := cmd.Output()
	if err != nil {
		return nil // docker not running, skip
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		snap, err := ParseDockerPSLine(line)
		if err != nil {
			continue
		}
		if err := s.SaveDockerSnapshot(ctx, syncID, snap); err != nil {
			return err
		}
	}

	return nil
}

type dockerPSOutput struct {
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	Status string `json:"Status"`
	Ports  string `json:"Ports"`
}

// ParseDockerPSLine parses a single JSON line from docker ps output.
func ParseDockerPSLine(line string) (domain.DockerSnapshot, error) {
	var ps dockerPSOutput
	if err := json.Unmarshal([]byte(line), &ps); err != nil {
		return domain.DockerSnapshot{}, err
	}
	return domain.DockerSnapshot{
		ContainerName: ps.Names,
		Image:         ps.Image,
		Status:        ps.Status,
		Ports:         ps.Ports,
	}, nil
}

func init() {
	Register(&DockerCollector{})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/collector/... -run TestDocker -v`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/collector/docker.go internal/collector/docker_test.go
git commit -m "feat: add docker status collector with JSON parsing"
```

---

## Task 10: GitHub Notifications Collector

**Files:**
- Create: `internal/collector/github.go`
- Create: `internal/collector/github_test.go`

- [ ] **Step 1: Write github collector tests**

`internal/collector/github_test.go`:
```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/collector/... -run TestGitHub -v`
Expected: Compilation error

- [ ] **Step 3: Implement github collector**

`internal/collector/github.go`:
```go
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

func (g *GitHubCollector) Name() string        { return "github" }
func (g *GitHubCollector) EnvVars() []string   { return []string{"GITHUB_TOKEN"} }

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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/collector/... -run TestGitHub -v`
Expected: All 3 tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/collector/github.go internal/collector/github_test.go
git commit -m "feat: add GitHub notifications collector with API parsing"
```

---

## Task 11: Sync Engine

**Files:**
- Create: `internal/sync/engine.go`
- Create: `internal/sync/engine_test.go`

- [ ] **Step 1: Write sync engine tests**

`internal/sync/engine_test.go`:
```go
package sync_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
	psync "github.com/xcoleman/pulse/internal/sync"
)

type fakeCollector struct {
	name   string
	err    error
	called bool
}

func (f *fakeCollector) Name() string                                                    { return f.name }
func (f *fakeCollector) EnvVars() []string                                               { return nil }
func (f *fakeCollector) Enabled(cfg *config.Config) bool                                 { return true }
func (f *fakeCollector) Collect(ctx context.Context, s store.Store, cfg *config.Config, syncID int64) error {
	f.called = true
	return f.err
}

func newTestStoreForSync(t *testing.T) store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSyncEngine_AllSuccess(t *testing.T) {
	s := newTestStoreForSync(t)
	cfg := &config.Config{Sync: config.SyncConfig{Timeout: "30s"}}

	c1 := &fakeCollector{name: "fake1"}
	c2 := &fakeCollector{name: "fake2"}

	engine := psync.NewEngine(s, cfg)
	result := engine.Run(context.Background(), []collector.Collector{c1, c2})

	if result.Status != "success" {
		t.Errorf("expected success, got %s", result.Status)
	}
	if !c1.called || !c2.called {
		t.Error("expected both collectors to be called")
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %v", result.Errors)
	}
}

func TestSyncEngine_PartialFailure(t *testing.T) {
	s := newTestStoreForSync(t)
	cfg := &config.Config{Sync: config.SyncConfig{Timeout: "30s"}}

	c1 := &fakeCollector{name: "ok"}
	c2 := &fakeCollector{name: "fail", err: errors.New("boom")}

	engine := psync.NewEngine(s, cfg)
	result := engine.Run(context.Background(), []collector.Collector{c1, c2})

	if result.Status != "partial" {
		t.Errorf("expected partial, got %s", result.Status)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestSyncEngine_TotalFailure(t *testing.T) {
	s := newTestStoreForSync(t)
	cfg := &config.Config{Sync: config.SyncConfig{Timeout: "30s"}}

	c1 := &fakeCollector{name: "fail1", err: errors.New("boom1")}
	c2 := &fakeCollector{name: "fail2", err: errors.New("boom2")}

	engine := psync.NewEngine(s, cfg)
	result := engine.Run(context.Background(), []collector.Collector{c1, c2})

	if result.Status != "failed" {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if len(result.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors))
	}
}

func TestSyncEngine_OnlyMode(t *testing.T) {
	s := newTestStoreForSync(t)
	cfg := &config.Config{Sync: config.SyncConfig{Timeout: "30s"}}

	c1 := &fakeCollector{name: "git"}
	c2 := &fakeCollector{name: "docker"}

	engine := psync.NewEngine(s, cfg)
	result := engine.RunOnly(context.Background(), []collector.Collector{c1, c2}, "git")

	if result.Status != "success" {
		t.Errorf("expected success, got %s", result.Status)
	}
	if !c1.called {
		t.Error("expected git collector to be called")
	}
	if c2.called {
		t.Error("expected docker collector NOT to be called")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/sync/... -v`
Expected: Compilation error

- [ ] **Step 3: Implement sync engine**

`internal/sync/engine.go`:
```go
package sync

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
)

type Result struct {
	SyncID int64
	Status string // success, partial, failed
	Errors []string
}

type Engine struct {
	store store.Store
	cfg   *config.Config
}

func NewEngine(s store.Store, cfg *config.Config) *Engine {
	return &Engine{store: s, cfg: cfg}
}

func (e *Engine) Run(ctx context.Context, collectors []collector.Collector) Result {
	return e.runCollectors(ctx, collectors)
}

func (e *Engine) RunOnly(ctx context.Context, collectors []collector.Collector, only string) Result {
	var filtered []collector.Collector
	for _, c := range collectors {
		if c.Name() == only {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return Result{Status: "failed", Errors: []string{fmt.Sprintf("collector %q not found", only)}}
	}
	return e.runCollectors(ctx, filtered)
}

func (e *Engine) runCollectors(ctx context.Context, collectors []collector.Collector) Result {
	syncID, err := e.store.CreateSyncRun(ctx)
	if err != nil {
		return Result{Status: "failed", Errors: []string{fmt.Sprintf("creating sync run: %v", err)}}
	}

	timeout := parseDuration(e.cfg.Sync.Timeout, 30*time.Second)

	var errs []string
	for _, c := range collectors {
		cCtx, cancel := context.WithTimeout(ctx, timeout)
		if err := c.Collect(cCtx, e.store, e.cfg, syncID); err != nil {
			log.Printf("WARN: collector %q failed: %v", c.Name(), err)
			errs = append(errs, fmt.Sprintf("%s: %v", c.Name(), err))
		}
		cancel()
	}

	// Prune old briefing history (30 days)
	e.store.PruneBriefingHistory(ctx, time.Now().Add(-30*24*time.Hour))

	var status string
	switch {
	case len(errs) == 0:
		status = "success"
	case len(errs) < len(collectors):
		status = "partial"
	default:
		status = "failed"
	}

	errMsg := strings.Join(errs, "; ")
	e.store.CompleteSyncRun(ctx, syncID, status, errMsg)

	return Result{SyncID: syncID, Status: status, Errors: errs}
}

func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/sync/... -v`
Expected: All 4 tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/sync/ internal/collector/
git commit -m "feat: add sync engine with timeout, partial failure handling, and --only mode"
```

---

## Task 12: Briefing Engine

**Files:**
- Create: `internal/briefing/engine.go`
- Create: `internal/briefing/engine_test.go`

- [ ] **Step 1: Write briefing engine tests**

`internal/briefing/engine_test.go`:
```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/briefing/... -v`
Expected: Compilation error

- [ ] **Step 3: Implement briefing engine**

`internal/briefing/engine.go`:
```go
package briefing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

type Engine struct {
	store store.Store
	cfg   *config.Config
}

func NewEngine(s store.Store, cfg *config.Config) *Engine {
	return &Engine{store: s, cfg: cfg}
}

func (e *Engine) Build(ctx context.Context) (*domain.Briefing, error) {
	syncID, err := e.store.LatestSyncID(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting latest sync: %w", err)
	}
	if syncID == 0 {
		return nil, fmt.Errorf("no sync data available — run 'pulse sync' first")
	}

	b := &domain.Briefing{
		GeneratedAt: time.Now(),
	}

	// Projects
	snapshots, err := e.store.GetGitSnapshots(ctx, syncID)
	if err != nil {
		return nil, fmt.Errorf("reading git snapshots: %w", err)
	}
	for _, snap := range snapshots {
		branches, _ := e.store.GetGitBranches(ctx, syncID, snap.RepoPath)
		b.Projects = append(b.Projects, domain.ProjectSummary{
			GitSnapshot: snap,
			Branches:    branches,
		})
	}

	// Notifications
	b.Notifications, _ = e.store.GetGitHubNotifications(ctx, syncID)

	// Costs
	since := parsePeriod(e.cfg.Costs.DefaultPeriod)
	costEntries, _ := e.store.GetCostEntries(ctx, since)
	b.CostSummary = buildCostSummary(costEntries, e.cfg.Costs.Currency, e.cfg.Costs.DefaultPeriod, since)

	// Docker
	b.Docker, _ = e.store.GetDockerSnapshots(ctx, syncID)

	// System
	sys, _ := e.store.GetSystemSnapshot(ctx, syncID)
	if sys != nil {
		b.System = *sys
	}

	return b, nil
}

func buildCostSummary(entries []domain.CostEntry, currency, period string, since time.Time) domain.CostSummary {
	summary := domain.CostSummary{
		Currency: currency,
		Period:   period,
	}

	byService := make(map[string]*domain.ServiceCost)
	for _, e := range entries {
		sc, ok := byService[e.Service]
		if !ok {
			sc = &domain.ServiceCost{Service: e.Service, UsageUnit: e.UsageUnit}
			byService[e.Service] = sc
		}
		sc.AmountCents += e.AmountCents
		sc.UsageQuantity += e.UsageQuantity
		summary.TotalCents += e.AmountCents
	}

	for _, sc := range byService {
		summary.ByService = append(summary.ByService, *sc)
	}

	// Burn rate: total cents / days in period
	days := time.Since(since).Hours() / 24
	if days > 0 {
		summary.BurnRateCents = int(float64(summary.TotalCents) / days)
	}

	return summary
}

func parsePeriod(period string) time.Time {
	period = strings.TrimSpace(period)
	if period == "" {
		period = "30d"
	}

	// Parse "Nd" format
	if strings.HasSuffix(period, "d") {
		numStr := strings.TrimSuffix(period, "d")
		var days int
		fmt.Sscanf(numStr, "%d", &days)
		if days > 0 {
			return time.Now().Add(-time.Duration(days) * 24 * time.Hour)
		}
	}

	// Default: 30 days
	return time.Now().Add(-30 * 24 * time.Hour)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/briefing/... -v`
Expected: Both tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/briefing/
git commit -m "feat: add briefing engine with cost summary aggregation"
```

---

## Task 13: Stdout Writer

**Files:**
- Create: `internal/writer/registry.go`
- Create: `internal/writer/stdout.go`
- Create: `internal/writer/stdout_test.go`
- Create: `internal/writer/testdata/briefing_full.golden`

- [ ] **Step 1: Create writer registry**

`internal/writer/registry.go`:
```go
package writer

import (
	"context"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
)

// Writer renders a briefing to an output target.
type Writer interface {
	Name() string
	Write(ctx context.Context, briefing *domain.Briefing, cfg *config.Config) error
}
```

- [ ] **Step 2: Write stdout writer tests**

`internal/writer/stdout_test.go`:
```go
package writer_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/writer"
)

func sampleBriefing() *domain.Briefing {
	return &domain.Briefing{
		GeneratedAt: time.Date(2026, 3, 20, 7, 0, 0, 0, time.Local),
		Projects: []domain.ProjectSummary{
			{GitSnapshot: domain.GitSnapshot{RepoName: "cortex", Branch: "main", DirtyFiles: 3, Ahead: 2}},
			{GitSnapshot: domain.GitSnapshot{RepoName: "pulse", Branch: "main", DirtyFiles: 0, Ahead: 0}},
		},
		Notifications: []domain.Notification{
			{RepoName: "obsidian-mcp", Type: "pr", Title: "Fix FTS5 indexing", State: "open"},
		},
		CostSummary: domain.CostSummary{
			TotalCents: 1842, Currency: "USD", Period: "30d", BurnRateCents: 61,
			ByService: []domain.ServiceCost{
				{Service: "claude", AmountCents: 1482},
				{Service: "voyage", AmountCents: 210},
				{Service: "tavily", AmountCents: 150},
			},
		},
		System: domain.SystemSnapshot{
			CPUPct: 12.5, MemoryUsedMB: 18200, MemoryTotalMB: 32000,
			DiskUsedGB: 142, DiskTotalGB: 256,
		},
	}
}

func TestStdoutWriter_ContainsSections(t *testing.T) {
	var buf bytes.Buffer
	w := writer.NewStdoutWriter(&buf)
	cfg := &config.Config{}

	err := w.Write(context.Background(), sampleBriefing(), cfg)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	output := buf.String()

	// Check key sections are present
	sections := []string{"Projects", "GitHub", "Costs", "System"}
	for _, section := range sections {
		if !bytes.Contains([]byte(output), []byte(section)) {
			t.Errorf("expected output to contain section %q", section)
		}
	}

	// Check project data
	if !bytes.Contains([]byte(output), []byte("cortex")) {
		t.Error("expected output to contain 'cortex'")
	}
	if !bytes.Contains([]byte(output), []byte("pulse")) {
		t.Error("expected output to contain 'pulse'")
	}
}

func TestStdoutWriter_CostFormatting(t *testing.T) {
	var buf bytes.Buffer
	w := writer.NewStdoutWriter(&buf)
	cfg := &config.Config{}

	err := w.Write(context.Background(), sampleBriefing(), cfg)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("$14.82")) {
		t.Error("expected output to contain '$14.82' for claude costs")
	}
	if !bytes.Contains([]byte(output), []byte("$18.42")) {
		t.Error("expected output to contain '$18.42' for total")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/writer/... -v`
Expected: Compilation error

- [ ] **Step 4: Implement stdout writer**

`internal/writer/stdout.go`:
```go
package writer

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
)

type StdoutWriter struct {
	out io.Writer
}

func NewStdoutWriter(out io.Writer) *StdoutWriter {
	if out == nil {
		out = os.Stdout
	}
	return &StdoutWriter{out: out}
}

func (w *StdoutWriter) Name() string { return "stdout" }

func (w *StdoutWriter) Write(ctx context.Context, b *domain.Briefing, cfg *config.Config) error {
	fmt.Fprintf(w.out, "Pulse Briefing — %s\n\n", b.GeneratedAt.Format("Mon Jan 2, 2006"))

	w.writeProjects(b.Projects)
	w.writeNotifications(b.Notifications)
	w.writeCosts(b.CostSummary)
	w.writeDocker(b.Docker)
	w.writeSystem(b.System)

	return nil
}

func (w *StdoutWriter) writeProjects(projects []domain.ProjectSummary) {
	fmt.Fprintf(w.out, "--- Projects ---\n")
	for _, p := range projects {
		icon := "✓"
		details := "clean"
		if p.DirtyFiles > 0 || p.Ahead > 0 || p.Behind > 0 {
			icon = "⚠"
			var parts []string
			if p.DirtyFiles > 0 {
				parts = append(parts, fmt.Sprintf("%d dirty", p.DirtyFiles))
			}
			if p.Ahead > 0 {
				parts = append(parts, fmt.Sprintf("%d ahead", p.Ahead))
			}
			if p.Behind > 0 {
				parts = append(parts, fmt.Sprintf("%d behind", p.Behind))
			}
			details = strings.Join(parts, ", ")
		}
		fmt.Fprintf(w.out, "  %s %s (%s) — %s\n", icon, p.RepoName, p.Branch, details)
	}
	fmt.Fprintln(w.out)
}

func (w *StdoutWriter) writeNotifications(notifs []domain.Notification) {
	if len(notifs) == 0 {
		return
	}
	fmt.Fprintf(w.out, "--- GitHub ---\n")
	for _, n := range notifs {
		icon := "●"
		fmt.Fprintf(w.out, "  %s %s — %s [%s]\n", icon, n.RepoName, n.Title, n.Type)
	}
	fmt.Fprintln(w.out)
}

func (w *StdoutWriter) writeCosts(cs domain.CostSummary) {
	if cs.TotalCents == 0 {
		return
	}
	fmt.Fprintf(w.out, "--- Costs (%s) ---\n", cs.Period)
	for _, sc := range cs.ByService {
		fmt.Fprintf(w.out, "  %s: $%.2f\n", sc.Service, float64(sc.AmountCents)/100)
	}
	fmt.Fprintf(w.out, "  Total: $%.2f — Burn: $%.2f/day\n", float64(cs.TotalCents)/100, float64(cs.BurnRateCents)/100)
	fmt.Fprintln(w.out)
}

func (w *StdoutWriter) writeDocker(containers []domain.DockerSnapshot) {
	if len(containers) == 0 {
		return
	}
	fmt.Fprintf(w.out, "--- Docker ---\n")
	for _, c := range containers {
		fmt.Fprintf(w.out, "  %s (%s) — %s\n", c.ContainerName, c.Image, c.Status)
	}
	fmt.Fprintln(w.out)
}

func (w *StdoutWriter) writeSystem(sys domain.SystemSnapshot) {
	fmt.Fprintf(w.out, "--- System ---\n")
	fmt.Fprintf(w.out, "  CPU: %.0f%% — RAM: %.1f/%.1f GB — Disk: %.0f/%.0f GB\n",
		sys.CPUPct,
		sys.MemoryUsedMB/1024, sys.MemoryTotalMB/1024,
		sys.DiskUsedGB, sys.DiskTotalGB)
	fmt.Fprintln(w.out)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/writer/... -v`
Expected: Both tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/writer/
git commit -m "feat: add stdout writer with formatted briefing output"
```

---

## Task 14: Obsidian Writer

**Files:**
- Create: `internal/writer/obsidian.go`
- Create: `internal/writer/obsidian_test.go`

- [ ] **Step 1: Write obsidian writer tests**

`internal/writer/obsidian_test.go`:
```go
package writer_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/writer"
)

func TestObsidianWriter_CreatesSection(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "2026-03-20.md")

	// Create existing daily note
	existing := "# Daily Note\n\nSome content here.\n"
	os.WriteFile(notePath, []byte(existing), 0644)

	cfg := &config.Config{
		Obsidian: config.ObsidianConfig{
			VaultPath:      dir,
			DailyNotePath:  "YYYY-MM-DD.md",
			SectionHeading: "## Pulse Briefing",
		},
	}

	w := writer.NewObsidianWriter()
	b := sampleBriefing()

	err := w.Write(context.Background(), b, cfg)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	content, _ := os.ReadFile(notePath)
	if !strings.Contains(string(content), "## Pulse Briefing") {
		t.Error("expected briefing section heading")
	}
	if !strings.Contains(string(content), "cortex") {
		t.Error("expected project data in note")
	}
	if !strings.Contains(string(content), "Some content here.") {
		t.Error("expected existing content preserved")
	}
}

func TestObsidianWriter_MissingConfig(t *testing.T) {
	cfg := &config.Config{} // No obsidian config

	w := writer.NewObsidianWriter()
	err := w.Write(context.Background(), sampleBriefing(), cfg)

	if err == nil {
		t.Error("expected error for missing obsidian config")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/writer/... -run TestObsidian -v`
Expected: Compilation error

- [ ] **Step 3: Implement obsidian writer**

`internal/writer/obsidian.go`:
```go
package writer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
)

type ObsidianWriter struct{}

func NewObsidianWriter() *ObsidianWriter {
	return &ObsidianWriter{}
}

func (w *ObsidianWriter) Name() string { return "obsidian" }

func (w *ObsidianWriter) Write(ctx context.Context, b *domain.Briefing, cfg *config.Config) error {
	if cfg.Obsidian.VaultPath == "" {
		return fmt.Errorf("obsidian vault_path not configured — set it in ~/.config/pulse/config.yaml")
	}

	notePath := cfg.ObsidianDailyNotePath(b.GeneratedAt)
	heading := cfg.Obsidian.SectionHeading
	if heading == "" {
		heading = "## Pulse Briefing"
	}

	// Render briefing as markdown
	var md bytes.Buffer
	stdoutWriter := NewStdoutWriter(&md)
	stdoutWriter.Write(ctx, b, cfg)

	section := fmt.Sprintf("\n%s\n\n%s\n", heading, md.String())

	// Read existing note or create new
	existing, err := os.ReadFile(notePath)
	if err != nil {
		dir := filepath.Dir(notePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating note directory: %w", err)
		}
		return os.WriteFile(notePath, []byte(section), 0644)
	}

	// Check if section already exists — replace it
	content := string(existing)
	if idx := strings.Index(content, heading); idx >= 0 {
		rest := content[idx+len(heading):]
		nextHeading := strings.Index(rest, "\n## ")
		if nextHeading >= 0 {
			content = content[:idx] + section + rest[nextHeading:]
		} else {
			content = content[:idx] + section
		}
	} else {
		content = content + section
	}

	return os.WriteFile(notePath, []byte(content), 0644)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/writer/... -run TestObsidian -v`
Expected: Both tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/writer/obsidian.go internal/writer/obsidian_test.go
git commit -m "feat: add Obsidian daily note writer with section append/replace"
```

---

## Task 15: CLI Commands

**Files:**
- Create: `internal/cli/sync_cmd.go`
- Create: `internal/cli/costs_cmd.go`
- Create: `internal/cli/projects_cmd.go`
- Create: `internal/cli/obsidian_cmd.go`
- Create: `internal/cli/config_cmd.go`
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Update root command to run briefing**

`internal/cli/root.go` — update `RunE` to load config, open store, build briefing, and write to stdout. Wire up the `--since` and `--json` flags.

```go
package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/briefing"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
	"github.com/xcoleman/pulse/internal/writer"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "pulse",
	Short: "Personal command center — briefing, project health, cost tracking",
	Long:  "Pulse synthesizes signals from your projects, AI services, and dev environment into a single morning briefing.",
	RunE:  runBriefing,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug output")
	rootCmd.PersistentFlags().Bool("json", false, "Output as JSON")
	rootCmd.Flags().String("since", "", "Show data since duration (e.g., 24h, 7d)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadConfig() (*config.Config, error) {
	return config.Load(config.DefaultConfigPath())
}

func openStore() (store.Store, error) {
	dbPath := config.DefaultConfigDir() + "/pulse.db"
	return store.NewSQLite(dbPath)
}

func runBriefing(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	s, err := openStore()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer s.Close()

	engine := briefing.NewEngine(s, cfg)
	b, err := engine.Build(cmd.Context())
	if err != nil {
		return err
	}

	jsonFlag, _ := cmd.Flags().GetBool("json")
	if jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(b)
	}

	w := writer.NewStdoutWriter(nil)
	return w.Write(cmd.Context(), b, cfg)
}
```

- [ ] **Step 2: Create sync command**

`internal/cli/sync_cmd.go`:
```go
package cli

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/collector"
	psync "github.com/xcoleman/pulse/internal/sync"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Collect data from all sources",
	RunE:  runSync,
}

func init() {
	syncCmd.Flags().String("only", "", "Run only a specific collector")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	s, err := openStore()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer s.Close()

	engine := psync.NewEngine(s, cfg)
	collectors := collector.Enabled(cfg)

	only, _ := cmd.Flags().GetString("only")

	var result psync.Result
	if only != "" {
		result = engine.RunOnly(cmd.Context(), collectors, only)
	} else {
		result = engine.Run(cmd.Context(), collectors)
	}

	for _, e := range result.Errors {
		log.Printf("WARN: %s", e)
	}

	fmt.Fprintf(os.Stderr, "sync: %s (run %d)\n", result.Status, result.SyncID)

	switch result.Status {
	case "success":
		return nil
	case "partial":
		os.Exit(1)
	default:
		os.Exit(2)
	}
	return nil
}
```

- [ ] **Step 3: Create costs command**

`internal/cli/costs_cmd.go`:
```go
package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/briefing"
)

var costsCmd = &cobra.Command{
	Use:   "costs",
	Short: "Print cost summary",
	RunE:  runCosts,
}

func init() {
	costsCmd.Flags().String("service", "", "Filter to a specific service")
	costsCmd.Flags().String("period", "", "Time period (e.g., 7d, 30d)")
	rootCmd.AddCommand(costsCmd)
}

func runCosts(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	period, _ := cmd.Flags().GetString("period")
	if period != "" {
		cfg.Costs.DefaultPeriod = period
	}

	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	engine := briefing.NewEngine(s, cfg)
	b, err := engine.Build(cmd.Context())
	if err != nil {
		return err
	}

	service, _ := cmd.Flags().GetString("service")
	cs := b.CostSummary

	jsonFlag, _ := cmd.Flags().GetBool("json")

	if service != "" {
		// Filter
		for _, sc := range cs.ByService {
			if sc.Service == service {
				if jsonFlag {
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(sc)
				}
				fmt.Printf("%s: $%.2f (%.0f %s)\n", sc.Service, float64(sc.AmountCents)/100, sc.UsageQuantity, sc.UsageUnit)
				return nil
			}
		}
		return fmt.Errorf("service %q not found", service)
	}

	if jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(cs)
	}

	fmt.Printf("Costs (%s)\n\n", cs.Period)
	for _, sc := range cs.ByService {
		fmt.Printf("  %s: $%.2f\n", sc.Service, float64(sc.AmountCents)/100)
	}
	fmt.Printf("\n  Total: $%.2f — Burn: $%.2f/day\n",
		float64(cs.TotalCents)/100, float64(cs.BurnRateCents)/100)

	return nil
}
```

- [ ] **Step 4: Create projects command**

`internal/cli/projects_cmd.go`:
```go
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/briefing"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Print project health summary",
	RunE:  runProjects,
}

func init() {
	projectsCmd.Flags().String("repo", "", "Filter to a specific repo")
	rootCmd.AddCommand(projectsCmd)
}

func runProjects(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	engine := briefing.NewEngine(s, cfg)
	b, err := engine.Build(cmd.Context())
	if err != nil {
		return err
	}

	repo, _ := cmd.Flags().GetString("repo")
	jsonFlag, _ := cmd.Flags().GetBool("json")

	projects := b.Projects
	if repo != "" {
		var filtered []interface{}
		for _, p := range projects {
			if p.RepoName == repo {
				if jsonFlag {
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(p)
				}
				printProject(p)
				return nil
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("repo %q not found", repo)
		}
	}

	if jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(projects)
	}

	for _, p := range projects {
		printProject(p)
	}
	return nil
}

func printProject(p interface{}) {
	// Type assert — this is a domain.ProjectSummary
	type projectLike interface {
		GetRepoName() string
	}
	// Simple formatting using fmt
	fmt.Printf("%v\n", p)
}
```

Actually, let me simplify `printProject` to work with the concrete type:

`internal/cli/projects_cmd.go` (corrected):
```go
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/briefing"
	"github.com/xcoleman/pulse/internal/domain"
)

// Note: strings import is used by printProject

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Print project health summary",
	RunE:  runProjects,
}

func init() {
	projectsCmd.Flags().String("repo", "", "Filter to a specific repo")
	rootCmd.AddCommand(projectsCmd)
}

func runProjects(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	engine := briefing.NewEngine(s, cfg)
	b, err := engine.Build(cmd.Context())
	if err != nil {
		return err
	}

	repo, _ := cmd.Flags().GetString("repo")
	jsonFlag, _ := cmd.Flags().GetBool("json")

	if repo != "" {
		for _, p := range b.Projects {
			if p.RepoName == repo {
				if jsonFlag {
					return jsonOut(p)
				}
				printProject(p)
				return nil
			}
		}
		return fmt.Errorf("repo %q not found", repo)
	}

	if jsonFlag {
		return jsonOut(b.Projects)
	}

	for _, p := range b.Projects {
		printProject(p)
	}
	return nil
}

func printProject(p domain.ProjectSummary) {
	icon := "✓"
	details := "clean"
	if p.DirtyFiles > 0 || p.Ahead > 0 || p.Behind > 0 {
		icon = "⚠"
		var parts []string
		if p.DirtyFiles > 0 {
			parts = append(parts, fmt.Sprintf("%d dirty", p.DirtyFiles))
		}
		if p.Ahead > 0 {
			parts = append(parts, fmt.Sprintf("%d ahead", p.Ahead))
		}
		if p.Behind > 0 {
			parts = append(parts, fmt.Sprintf("%d behind", p.Behind))
		}
		details = strings.Join(parts, ", ")
	}
	fmt.Printf("  %s %s (%s) — %s\n", icon, p.RepoName, p.Branch, details)

	if len(p.Branches) > 1 {
		for _, br := range p.Branches {
			if !br.IsCurrent {
				merged := ""
				if br.IsMerged {
					merged = " [merged]"
				}
				fmt.Printf("      ↳ %s%s\n", br.BranchName, merged)
			}
		}
	}
}

func jsonOut(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
```

- [ ] **Step 5: Create obsidian command**

`internal/cli/obsidian_cmd.go`:
```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/briefing"
	"github.com/xcoleman/pulse/internal/writer"
)

var obsidianCmd = &cobra.Command{
	Use:   "obsidian",
	Short: "Append briefing to today's Obsidian daily note",
	RunE:  runObsidian,
}

func init() {
	rootCmd.AddCommand(obsidianCmd)
}

func runObsidian(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	engine := briefing.NewEngine(s, cfg)
	b, err := engine.Build(cmd.Context())
	if err != nil {
		return err
	}

	w := writer.NewObsidianWriter()
	if err := w.Write(cmd.Context(), b, cfg); err != nil {
		return err
	}

	notePath := cfg.ObsidianDailyNotePath(b.GeneratedAt)
	fmt.Printf("Briefing written to %s\n", notePath)
	return nil
}
```

- [ ] **Step 6: Create config commands**

`internal/cli/config_cmd.go`:
```go
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Pulse configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate default config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.DefaultConfigPath()
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config already exists at %s", path)
		}
		if err := config.GenerateDefault(path); err != nil {
			return err
		}
		fmt.Printf("Config created at %s\n", path)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print current config",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.DefaultConfigPath()
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading config: %w (run 'pulse config init' to create)", err)
		}
		fmt.Print(string(data))
		return nil
	},
}

var configAdaptersCmd = &cobra.Command{
	Use:   "adapters",
	Short: "Show adapter status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		fmt.Printf("%-15s %-10s %-10s %s\n", "ADAPTER", "ENABLED", "ENV OK", "ENV VARS")
		fmt.Println(strings.Repeat("-", 60))

		for _, c := range collector.All() {
			enabled := cfg.AdapterEnabled(c.Name())
			enabledStr := "yes"
			if !enabled {
				enabledStr = "no"
			}

			envVars := c.EnvVars()
			envOK := "n/a"
			envList := "-"
			if len(envVars) > 0 {
				envList = strings.Join(envVars, ", ")
				allSet := true
				for _, v := range envVars {
					if os.Getenv(v) == "" {
						allSet = false
						break
					}
				}
				if allSet {
					envOK = "yes"
				} else {
					envOK = "MISSING"
				}
			}

			fmt.Printf("%-15s %-10s %-10s %s\n", c.Name(), enabledStr, envOK, envList)
		}
		return nil
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configAdaptersCmd)
	rootCmd.AddCommand(configCmd)
}
```

- [ ] **Step 7: Build and verify**

Run: `go build -o pulse ./cmd/pulse && ./pulse version`
Expected: `pulse dev`

Run: `./pulse config init` (if no config exists) or `./pulse --help`
Expected: Shows all subcommands

- [ ] **Step 8: Commit**

```bash
git add internal/cli/
git commit -m "feat: add all CLI commands (sync, costs, projects, obsidian, config)"
```

---

## Task 16: TUI — App Shell & Tab Navigation

**Files:**
- Create: `internal/tui/styles.go`
- Create: `internal/tui/app.go`
- Create: `internal/tui/briefing_tab.go`
- Create: `internal/tui/projects_tab.go`
- Create: `internal/tui/costs_tab.go`
- Create: `internal/tui/app_test.go`
- Create: `internal/cli/tui_cmd.go`

- [ ] **Step 1: Create shared styles**

`internal/tui/styles.go`:
```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	tabStyle = lipgloss.NewStyle().
			Padding(0, 2)

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			MarginBottom(1)

	sectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8FBC8F")).
			Bold(true)

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F0C674"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CC6666"))

	okStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8FBC8F"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444"))
)
```

- [ ] **Step 2: Create tab view stubs**

`internal/tui/briefing_tab.go`:
```go
package tui

import (
	"fmt"
	"strings"

	"github.com/xcoleman/pulse/internal/domain"
)

func renderBriefingTab(b *domain.Briefing, width int) string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render(fmt.Sprintf("Pulse Briefing — %s", b.GeneratedAt.Format("Mon Jan 2, 2006"))))
	sb.WriteString("\n\n")

	// Projects
	sb.WriteString(sectionStyle.Render("--- Projects ---"))
	sb.WriteString("\n")
	for _, p := range b.Projects {
		icon := okStyle.Render("✓")
		details := "clean"
		if p.DirtyFiles > 0 || p.Ahead > 0 || p.Behind > 0 {
			icon = warnStyle.Render("⚠")
			var parts []string
			if p.DirtyFiles > 0 {
				parts = append(parts, fmt.Sprintf("%d dirty", p.DirtyFiles))
			}
			if p.Ahead > 0 {
				parts = append(parts, fmt.Sprintf("%d ahead", p.Ahead))
			}
			if p.Behind > 0 {
				parts = append(parts, fmt.Sprintf("%d behind", p.Behind))
			}
			details = strings.Join(parts, ", ")
		}
		sb.WriteString(fmt.Sprintf("  %s %s (%s) — %s\n", icon, p.RepoName, p.Branch, details))
	}

	sb.WriteString("\n")

	// Notifications
	if len(b.Notifications) > 0 {
		sb.WriteString(sectionStyle.Render("--- GitHub ---"))
		sb.WriteString("\n")
		for _, n := range b.Notifications {
			sb.WriteString(fmt.Sprintf("  ● %s — %s [%s]\n", n.RepoName, n.Title, n.Type))
		}
		sb.WriteString("\n")
	}

	// Costs
	if b.CostSummary.TotalCents > 0 {
		sb.WriteString(sectionStyle.Render(fmt.Sprintf("--- Costs (%s) ---", b.CostSummary.Period)))
		sb.WriteString("\n")
		for _, sc := range b.CostSummary.ByService {
			sb.WriteString(fmt.Sprintf("  %s: $%.2f\n", sc.Service, float64(sc.AmountCents)/100))
		}
		sb.WriteString(fmt.Sprintf("  Total: $%.2f — Burn: $%.2f/day\n",
			float64(b.CostSummary.TotalCents)/100, float64(b.CostSummary.BurnRateCents)/100))
		sb.WriteString("\n")
	}

	// System
	sb.WriteString(sectionStyle.Render("--- System ---"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  CPU: %.0f%% — RAM: %.1f/%.1f GB — Disk: %.0f/%.0f GB\n",
		b.System.CPUPct,
		b.System.MemoryUsedMB/1024, b.System.MemoryTotalMB/1024,
		b.System.DiskUsedGB, b.System.DiskTotalGB))

	return sb.String()
}
```

`internal/tui/projects_tab.go`:
```go
package tui

import (
	"fmt"
	"strings"

	"github.com/xcoleman/pulse/internal/domain"
)

func renderProjectsTab(projects []domain.ProjectSummary, selected int, detail bool, width int) string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Projects"))
	sb.WriteString("\n\n")

	for i, p := range projects {
		icon := okStyle.Render("✓")
		if p.DirtyFiles > 0 || p.Ahead > 0 || p.Behind > 0 {
			icon = warnStyle.Render("⚠")
		}

		cursor := "  "
		if i == selected {
			cursor = "> "
		}

		sb.WriteString(fmt.Sprintf("%s%s %s (%s)", cursor, icon, p.RepoName, p.Branch))

		if p.DirtyFiles > 0 {
			sb.WriteString(fmt.Sprintf(" — %d dirty", p.DirtyFiles))
		}
		if p.Ahead > 0 {
			sb.WriteString(fmt.Sprintf(", %d ahead", p.Ahead))
		}
		sb.WriteString("\n")

		// Show detail if selected and in detail mode
		if i == selected && detail {
			sb.WriteString(fmt.Sprintf("      Last commit: %s — %s\n", p.LastCommitHash, p.LastCommitMsg))
			if len(p.Branches) > 0 {
				sb.WriteString("      Branches:\n")
				for _, br := range p.Branches {
					merged := ""
					if br.IsMerged {
						merged = " [merged]"
					}
					current := " "
					if br.IsCurrent {
						current = "*"
					}
					sb.WriteString(fmt.Sprintf("        %s %s%s\n", current, br.BranchName, merged))
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
```

`internal/tui/costs_tab.go`:
```go
package tui

import (
	"fmt"
	"strings"

	"github.com/xcoleman/pulse/internal/domain"
)

func renderCostsTab(cs domain.CostSummary, selected int, detail bool, width int) string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render(fmt.Sprintf("Costs (%s)", cs.Period)))
	sb.WriteString("\n\n")

	if cs.TotalCents == 0 {
		sb.WriteString(dimStyle.Render("  No cost data available"))
		return sb.String()
	}

	for i, sc := range cs.ByService {
		cursor := "  "
		if i == selected {
			cursor = "> "
		}

		// Bar chart
		barWidth := 20
		pct := float64(sc.AmountCents) / float64(cs.TotalCents)
		filled := int(pct * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		sb.WriteString(fmt.Sprintf("%s%-12s $%7.2f %s\n", cursor, sc.Service, float64(sc.AmountCents)/100, bar))
	}

	sb.WriteString(fmt.Sprintf("\n  Total: $%.2f — Burn: $%.2f/day\n",
		float64(cs.TotalCents)/100, float64(cs.BurnRateCents)/100))

	return sb.String()
}
```

- [ ] **Step 3: Create main TUI app model**

`internal/tui/app.go`:
```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xcoleman/pulse/internal/domain"
)

type tab int

const (
	tabBriefing tab = iota
	tabProjects
	tabCosts
)

type Model struct {
	briefing     *domain.Briefing
	activeTab    tab
	width        int
	height       int
	projSelected int
	projDetail   bool
	costSelected int
	costDetail   bool
}

func NewModel(b *domain.Briefing) Model {
	return Model{
		briefing:  b,
		activeTab: tabBriefing,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.activeTab = tabBriefing
		case "2":
			m.activeTab = tabProjects
			m.projDetail = false
		case "3":
			m.activeTab = tabCosts
			m.costDetail = false
		case "j", "down":
			m.moveDown()
		case "k", "up":
			m.moveUp()
		case "enter":
			m.toggleDetail()
		case "esc":
			m.closeDetail()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *Model) moveDown() {
	switch m.activeTab {
	case tabProjects:
		if m.projSelected < len(m.briefing.Projects)-1 {
			m.projSelected++
		}
	case tabCosts:
		if m.costSelected < len(m.briefing.CostSummary.ByService)-1 {
			m.costSelected++
		}
	}
}

func (m *Model) moveUp() {
	switch m.activeTab {
	case tabProjects:
		if m.projSelected > 0 {
			m.projSelected--
		}
	case tabCosts:
		if m.costSelected > 0 {
			m.costSelected--
		}
	}
}

func (m *Model) toggleDetail() {
	switch m.activeTab {
	case tabProjects:
		m.projDetail = !m.projDetail
	case tabCosts:
		m.costDetail = !m.costDetail
	}
}

func (m *Model) closeDetail() {
	m.projDetail = false
	m.costDetail = false
}

func (m Model) View() string {
	var sb strings.Builder

	// Tab bar
	tabs := []struct {
		label string
		key   string
		t     tab
	}{
		{"Briefing", "1", tabBriefing},
		{"Projects", "2", tabProjects},
		{"Costs", "3", tabCosts},
	}

	var tabParts []string
	for _, t := range tabs {
		label := fmt.Sprintf("%s %s", t.key, t.label)
		if t.t == m.activeTab {
			tabParts = append(tabParts, activeTabStyle.Render(label))
		} else {
			tabParts = append(tabParts, tabStyle.Render(label))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabParts...)
	sb.WriteString(tabBar)
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", m.width))
	sb.WriteString("\n\n")

	// Content
	switch m.activeTab {
	case tabBriefing:
		sb.WriteString(renderBriefingTab(m.briefing, m.width))
	case tabProjects:
		sb.WriteString(renderProjectsTab(m.briefing.Projects, m.projSelected, m.projDetail, m.width))
	case tabCosts:
		sb.WriteString(renderCostsTab(m.briefing.CostSummary, m.costSelected, m.costDetail, m.width))
	}

	// Help
	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("q quit · 1-3 tabs · j/k scroll · enter drill · esc back · ? help"))

	return sb.String()
}
```

- [ ] **Step 4: Write TUI tests**

`internal/tui/app_test.go`:
```go
package tui_test

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/tui"
)

func sampleBriefing() *domain.Briefing {
	return &domain.Briefing{
		GeneratedAt: time.Date(2026, 3, 20, 7, 0, 0, 0, time.Local),
		Projects: []domain.ProjectSummary{
			{GitSnapshot: domain.GitSnapshot{RepoName: "cortex", Branch: "main", DirtyFiles: 3, Ahead: 2}},
			{GitSnapshot: domain.GitSnapshot{RepoName: "pulse", Branch: "main"}},
		},
		CostSummary: domain.CostSummary{
			TotalCents: 1842, Currency: "USD", Period: "30d", BurnRateCents: 61,
			ByService: []domain.ServiceCost{
				{Service: "claude", AmountCents: 1482},
				{Service: "voyage", AmountCents: 210},
			},
		},
		System: domain.SystemSnapshot{CPUPct: 12.5, MemoryUsedMB: 18200, MemoryTotalMB: 32000},
	}
}

func TestTabSwitching(t *testing.T) {
	m := tui.NewModel(sampleBriefing())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Default is briefing tab
	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}

	// Switch to projects
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	view = m.View()
	if view == "" {
		t.Fatal("expected non-empty projects view")
	}

	// Switch to costs
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	view = m.View()
	if view == "" {
		t.Fatal("expected non-empty costs view")
	}
}

func TestProjectNavigation(t *testing.T) {
	m := tui.NewModel(sampleBriefing())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Switch to projects
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})

	// Move down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	// Enter detail
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Esc to close
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Should not panic
	_ = m.View()
}

func TestQuit(t *testing.T) {
	m := tui.NewModel(sampleBriefing())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if cmd == nil {
		t.Error("expected quit command")
	}
}
```

- [ ] **Step 5: Create TUI CLI command**

`internal/cli/tui_cmd.go`:
```go
package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/briefing"
	"github.com/xcoleman/pulse/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive dashboard",
	RunE:  runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	engine := briefing.NewEngine(s, cfg)
	b, err := engine.Build(cmd.Context())
	if err != nil {
		return err
	}

	model := tui.NewModel(b)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
```

- [ ] **Step 6: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass

- [ ] **Step 7: Build final binary and verify**

Run: `go build -o pulse ./cmd/pulse && ./pulse --help`
Expected: Shows all subcommands including tui

- [ ] **Step 8: Commit**

```bash
git add internal/tui/ internal/cli/tui_cmd.go
git commit -m "feat: add Bubble Tea TUI with tab navigation, drill-down, and keyboard controls"
```

---

## Task 17: Cost Collector Stubs

**Files:**
- Create: `internal/collector/cost_stub.go`
- Create: `internal/collector/cost_stub_test.go`

- [ ] **Step 1: Write cost stub tests**

`internal/collector/cost_stub_test.go`:
```go
package collector_test

import (
	"os"
	"testing"

	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
)

func TestCostCollectors_Registered(t *testing.T) {
	// Verify all cost collectors exist in the registry
	for _, name := range []string{"claude", "voyage", "tavily", "elevenlabs"} {
		_, ok := collector.Get(name)
		if !ok {
			t.Errorf("expected cost collector %q to be registered", name)
		}
	}
}

func TestCostCollectors_EnvVars(t *testing.T) {
	tests := []struct {
		name   string
		envVar string
	}{
		{"claude", "ANTHROPIC_API_KEY"},
		{"voyage", "VOYAGE_API_KEY"},
		{"tavily", "TAVILY_API_KEY"},
		{"elevenlabs", "ELEVENLABS_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := collector.Get(tt.name)
			envVars := c.EnvVars()
			if len(envVars) != 1 || envVars[0] != tt.envVar {
				t.Errorf("expected [%s], got %v", tt.envVar, envVars)
			}
		})
	}
}

func TestCostCollectors_DisabledWithoutEnvVar(t *testing.T) {
	cfg := &config.Config{}

	for _, name := range []string{"claude", "voyage", "tavily", "elevenlabs"} {
		c, _ := collector.Get(name)
		// Ensure env var is unset
		for _, ev := range c.EnvVars() {
			os.Unsetenv(ev)
		}
		if c.Enabled(cfg) {
			t.Errorf("expected %s collector disabled without env var", name)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/collector/... -run TestCost -v`
Expected: Compilation error

- [ ] **Step 3: Implement cost collector stubs**

`internal/collector/cost_stub.go`:
```go
package collector

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
)

// costStub is a placeholder collector for cost services whose billing APIs
// need to be verified before full implementation. Each stub registers itself,
// checks for its env var, and logs a warning when Collect is called.
type costStub struct {
	name   string
	envVar string
}

func (c *costStub) Name() string        { return c.name }
func (c *costStub) EnvVars() []string   { return []string{c.envVar} }

func (c *costStub) Enabled(cfg *config.Config) bool {
	if !cfg.AdapterEnabled(c.name) {
		return false
	}
	return os.Getenv(c.envVar) != ""
}

func (c *costStub) Collect(ctx context.Context, s store.Store, cfg *config.Config, syncID int64) error {
	// TODO: Implement actual API call once billing endpoint is verified
	log.Printf("WARN: %s cost collector is a stub — billing API integration pending", c.name)
	return fmt.Errorf("%s cost collector not yet implemented", c.name)
}

func init() {
	Register(&costStub{name: "claude", envVar: "ANTHROPIC_API_KEY"})
	Register(&costStub{name: "voyage", envVar: "VOYAGE_API_KEY"})
	Register(&costStub{name: "tavily", envVar: "TAVILY_API_KEY"})
	Register(&costStub{name: "elevenlabs", envVar: "ELEVENLABS_API_KEY"})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/collector/... -run TestCost -v`
Expected: All 3 tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/collector/cost_stub.go internal/collector/cost_stub_test.go
git commit -m "feat: add stub cost collectors for Claude, Voyage, Tavily, ElevenLabs"
```

---

## Task 18: Wire --since Flag & Save Briefing History

**Files:**
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/obsidian_cmd.go`
- Modify: `internal/briefing/engine.go`

- [ ] **Step 1: Update briefing engine to accept a `since` parameter**

Add to `internal/briefing/engine.go`:
```go
// BuildOptions configures briefing generation.
type BuildOptions struct {
	Since time.Time // override briefing time window
}

func (e *Engine) BuildWithOptions(ctx context.Context, opts BuildOptions) (*domain.Briefing, error) {
	// If no explicit since, use last briefing time
	since := opts.Since
	if since.IsZero() {
		lastBriefing, _ := e.store.GetLastBriefingTime(ctx)
		if !lastBriefing.IsZero() {
			since = lastBriefing
		}
	}

	// Rest of Build() logic...
	// (refactor Build() to call BuildWithOptions with zero options)
	return e.build(ctx, since)
}

func (e *Engine) Build(ctx context.Context) (*domain.Briefing, error) {
	return e.BuildWithOptions(ctx, BuildOptions{})
}
```

- [ ] **Step 2: Wire --since in root command**

Update `runBriefing` in `internal/cli/root.go`:
```go
	sinceStr, _ := cmd.Flags().GetString("since")
	var opts briefing.BuildOptions
	if sinceStr != "" {
		d, err := parseSinceDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
		opts.Since = time.Now().Add(-d)
	}

	b, err := engine.BuildWithOptions(cmd.Context(), opts)
```

Add helper:
```go
func parseSinceDuration(s string) (time.Duration, error) {
	// Support "Nd" format (e.g., "7d" → 168h)
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(numStr, "%d", &days); err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
```

- [ ] **Step 3: Save briefing history after rendering**

Update `runBriefing` in `internal/cli/root.go` to save after writing:
```go
	// Save to briefing history
	var rendered bytes.Buffer
	w := writer.NewStdoutWriter(&rendered)
	w.Write(cmd.Context(), b, cfg)

	// Write to stdout
	fmt.Print(rendered.String())

	// Save history
	s.SaveBriefing(cmd.Context(), domain.BriefingEntry{
		CreatedAt: time.Now(),
		Content:   rendered.String(),
		Writer:    "stdout",
	})
```

Do the same in `runObsidian`:
```go
	s.SaveBriefing(cmd.Context(), domain.BriefingEntry{
		CreatedAt: time.Now(),
		Content:   b.GeneratedAt.String(), // or rendered content
		Writer:    "obsidian",
	})
```

- [ ] **Step 4: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/cli/root.go internal/cli/obsidian_cmd.go internal/briefing/engine.go
git commit -m "feat: wire --since flag and save briefing history after rendering"
```

---

## Task 19: Integration Test — Full Flow

**Files:**
- Create: `internal/integration_test.go`

- [ ] **Step 1: Write end-to-end integration test**

`internal/integration_test.go`:
```go
package internal_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xcoleman/pulse/internal/briefing"
	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
	psync "github.com/xcoleman/pulse/internal/sync"
	"github.com/xcoleman/pulse/internal/writer"
	"os"
	"bytes"
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
}
```

- [ ] **Step 2: Run integration test**

Run: `go test ./internal/ -run TestFullSyncAndBriefing -v`
Expected: PASS

- [ ] **Step 3: Run full test suite**

Run: `go test ./... -v -count=1`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/integration_test.go
git commit -m "test: add full flow integration test (sync → briefing → stdout)"
```

---

## Task 20: Final Polish & Verify

- [ ] **Step 1: Add .gitignore**

Create `.gitignore`:
```
pulse
*.db
.superpowers/
```

- [ ] **Step 2: Run go vet and go mod tidy**

Run: `go vet ./... && go mod tidy`
Expected: No issues

- [ ] **Step 3: Final build**

Run: `go build -o pulse ./cmd/pulse`
Expected: Binary created

- [ ] **Step 4: Verify all commands work**

Run:
```bash
./pulse version
./pulse --help
./pulse config init  # if not already done
./pulse config show
./pulse config adapters
```
Expected: All commands produce output without errors

- [ ] **Step 5: Commit**

```bash
git add .gitignore go.mod go.sum
git commit -m "chore: add .gitignore, tidy modules"
```
