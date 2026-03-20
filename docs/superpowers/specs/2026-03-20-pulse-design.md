# Pulse — Design Specification

**Date:** 2026-03-20
**Status:** Draft
**Author:** xcoleman + Claude

## Overview

Pulse is a personal command center CLI tool that answers "what should I pay attention to right now?" It combines three concerns into one Go binary: morning briefing, project health monitoring, and AI service cost tracking.

The morning briefing is the primary interface. It synthesizes signals from project health and cost tracking into a single composable output. Project health and cost tracking are detail views you drill into when needed.

## Goals

- One command (`pulse`) gives you the morning briefing — what's changed, what needs attention, what it's costing you
- CLI-first: stdout output is scriptable, pipeable, and cron-friendly
- Optional TUI (`pulse tui`) for interactive exploration with one level of drill-down
- Obsidian integration: append briefings to your daily note
- Pluggable adapters: add/remove data sources without changing core code
- Single Go binary, installable globally, runs from any directory

## Non-Goals

- Not a web dashboard (Cortex, Corvade, Wraith already serve that role)
- Not a replacement for `git`, `gh`, or billing dashboards — Pulse tells you where to look, not everything about what's there
- Not a daemon — data collection runs via cron, reads are instant from SQLite

## Architecture

Single Go binary with three layers:

### Entry Points

| Command | Purpose |
|---------|---------|
| `pulse` | Print morning briefing to stdout |
| `pulse tui` | Launch interactive Bubble Tea dashboard |
| `pulse sync` | Collect all data, write to SQLite (cron target) |
| `pulse sync --only <adapter>` | Run a single collector |
| `pulse obsidian` | Append briefing to today's daily note |
| `pulse costs` | Print cost summary (filterable by `--service`, `--period`) |
| `pulse projects` | Print project health summary (filterable by `--repo`) |
| `pulse config init` | Generate default config file |
| `pulse config show` | Print current config |
| `pulse config adapters` | Show adapter status: enabled, disabled, or missing env var |
| `pulse version` | Print version |

All stdout commands support `--json` for scripting. Built with Cobra.

### Core Components

- **Briefing Engine** — reads from SQLite, composes a `Briefing` struct, dispatches to the appropriate Writer
- **Sync Engine** — orchestrates all enabled Collectors during `pulse sync`, handles per-adapter timeouts, logs warnings on failure but continues with remaining adapters
- **Config Manager** — reads `~/.config/pulse/config.yaml` for settings, environment variables for secrets
- **SQLite Store** — `~/.config/pulse/pulse.db` holds all collected data and history

### Interfaces

```go
// Collector gathers data from an external source and writes it to the store.
type Collector interface {
    Name() string                        // adapter key: "git", "claude", etc.
    EnvVars() []string                   // required env vars, e.g. ["ANTHROPIC_API_KEY"]
    Enabled(cfg Config) bool             // check config + env vars
    Collect(ctx context.Context, store *Store, cfg Config) error
}

// Writer renders a briefing to an output target.
type Writer interface {
    Name() string                        // "stdout", "tui", "obsidian"
    Write(ctx context.Context, briefing *Briefing, cfg Config) error
}
```

All adapters self-register at init via a global registry. Adding a new adapter = one file implementing the interface + one registration line.

### Briefing Struct

The intermediate representation between the DB and Writers:

```go
type Briefing struct {
    GeneratedAt    time.Time
    Projects       []ProjectSummary
    Notifications  []Notification
    CostSummary    CostSummary
    Docker         []ContainerStatus
    System         SystemStatus
}
```

Each Writer decides how to render it: stdout formats as plain text, TUI renders as Bubble Tea components, Obsidian writes as markdown.

### Data Flow

```
Collectors → Sync Engine → SQLite → Briefing Engine → Writers
```

- `pulse sync` (cron, hourly) runs Collectors, writes to SQLite
- `pulse` / `pulse tui` / `pulse obsidian` reads from SQLite, builds Briefing, dispatches to Writer

## Collectors (v1)

### Must-Have

| Collector | Source | Data |
|-----------|--------|------|
| Git Scanner | Local repos | Branch, dirty files, ahead/behind, last commit, stale branches |
| GitHub API | GitHub REST API | PRs, issues, CI failures |
| Claude Costs | Anthropic API | Usage and spend |
| System Resources | `/proc`, `free`, `df` | CPU, RAM, disk usage |
| Docker Status | Docker CLI/API | Running containers, resource usage |

### If API Exists

| Collector | Source | Data |
|-----------|--------|------|
| Voyage AI | Voyage API | Usage and spend |
| Tavily | Tavily API | Usage and spend |
| ElevenLabs | ElevenLabs API | Usage and spend |

### v1.1

| Collector | Source | Data |
|-----------|--------|------|
| Ollama Compute | Local instrumentation | Model usage, compute time |

## Writers (v1)

| Writer | Target | Format |
|--------|--------|--------|
| stdout | Terminal | Formatted plain text with color |
| TUI | Terminal | Bubble Tea interactive dashboard |
| Obsidian | Vault daily note | Markdown appended under configurable heading |

## Project Discovery

Auto-scan with overrides:

```yaml
projects:
  scan:
    - ~/projects-wsl
  ignore:
    - voidterm-builds
    - docs
```

Pulse recursively finds git repos under `scan` directories, excluding entries in `ignore`.

## Data Collection

- `pulse sync` runs all enabled Collectors, called by cron hourly (`0 * * * *`)
- Each Collector runs with a configurable timeout (default: 30s)
- Failed Collectors log a warning; sync continues with remaining adapters
- Exit codes: 0 (all succeeded), 1 (partial — some failed), 2 (total failure)
- `pulse sync --only <adapter>` runs a single Collector for testing/debugging

## Credential Management

- API keys read from environment variables only (e.g., `ANTHROPIC_API_KEY`)
- No secrets stored in config files — config is safe to version control
- `pulse config adapters` shows which env vars are set/missing for each adapter
- Missing env var = adapter skipped with warning, not a hard failure

## Data Model

SQLite database at `~/.config/pulse/pulse.db`.

### sync_runs

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| started_at | DATETIME | Sync start time |
| completed_at | DATETIME | Sync end time |
| status | TEXT | success, partial, failed |
| error | TEXT | Error message if failed |

### git_snapshots

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| sync_id | INTEGER FK | References sync_runs |
| repo_path | TEXT | Absolute path to repo |
| repo_name | TEXT | Directory name |
| branch | TEXT | Current branch |
| dirty_files | INTEGER | Count of dirty files |
| ahead | INTEGER | Commits ahead of remote |
| behind | INTEGER | Commits behind remote |
| last_commit_hash | TEXT | Short hash |
| last_commit_msg | TEXT | Commit message |
| last_commit_at | DATETIME | Commit timestamp |

### git_branches

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| sync_id | INTEGER FK | References sync_runs |
| repo_path | TEXT | Absolute path to repo |
| branch_name | TEXT | Branch name |
| last_commit_at | DATETIME | Last commit on branch |
| is_merged | BOOLEAN | Whether branch is merged |
| is_current | BOOLEAN | Whether branch is checked out |

### github_notifications

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| sync_id | INTEGER FK | References sync_runs |
| repo_name | TEXT | Repository name |
| type | TEXT | pr, issue, ci |
| title | TEXT | Notification title |
| url | TEXT | Link to GitHub |
| state | TEXT | Current state |
| updated_at | DATETIME | Last updated |

### cost_entries

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| sync_id | INTEGER FK | References sync_runs |
| service | TEXT | claude, voyage, tavily, etc. |
| period_start | DATETIME | Billing period start |
| period_end | DATETIME | Billing period end |
| amount_cents | INTEGER | Cost in cents (avoids float) |
| currency | TEXT | USD |
| usage_quantity | REAL | Amount used |
| usage_unit | TEXT | tokens, searches, characters, etc. |
| raw_data | TEXT | JSON for service-specific details |

### docker_snapshots

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| sync_id | INTEGER FK | References sync_runs |
| container_name | TEXT | Container name |
| image | TEXT | Image name |
| status | TEXT | Running, stopped, etc. |
| ports | TEXT | JSON array of port mappings |
| cpu_pct | REAL | CPU usage percentage |
| memory_mb | REAL | Memory usage in MB |

### system_snapshots

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| sync_id | INTEGER FK | References sync_runs |
| cpu_pct | REAL | CPU usage percentage |
| memory_used_mb | REAL | Memory used in MB |
| memory_total_mb | REAL | Total memory in MB |
| disk_used_gb | REAL | Disk used in GB |
| disk_total_gb | REAL | Total disk in GB |

### briefing_history

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| created_at | DATETIME | When briefing was rendered |
| content | TEXT | Rendered markdown |
| writer | TEXT | stdout, obsidian, tui |

All collector tables link to `sync_id` for time-travel queries ("what did things look like at sync X?"). `briefing_history` is independent — written when a briefing is rendered, not during sync.

## TUI Layout

Tab bar layout with three full-screen views, switched with number keys:

| Tab | Key | Content |
|-----|-----|---------|
| Briefing | 1 | Default view — morning briefing with project alerts, GitHub notifications, cost summary, system status |
| Projects | 2 | All repos with status indicators. Enter to drill into a project: branches, recent commits, dirty files |
| Costs | 3 | Per-service totals with bar charts. Enter to drill into a service: daily/weekly breakdown |

**Navigation:**
- `1` / `2` / `3` — switch tabs
- `j` / `k` — scroll
- `Enter` — drill into selected item
- `Esc` — back to list
- `q` — quit
- `?` — help

**Drill-down depth:** One level. Overview → detail for a single item. Deeper investigation uses native tools.

## Config File

Located at `~/.config/pulse/config.yaml`, generated by `pulse config init`:

```yaml
# Pulse configuration
projects:
  scan:
    - ~/projects-wsl
  ignore:
    - voidterm-builds
    - docs

github:
  username: xcoleman

obsidian:
  vault_path: ~/path-to-your-vault
  daily_note_path: "Daily Notes/YYYY-MM-DD.md"
  section_heading: "## Pulse Briefing"

adapters:
  git: true
  github: true
  claude: true
  voyage: true
  tavily: true
  elevenlabs: true
  ollama: false        # v1.1
  docker: true
  system: true

sync:
  timeout: 30s         # per-adapter timeout

costs:
  default_period: 30d
  currency: USD
```

- No secrets in this file — API keys come from environment variables
- Adapters enabled by default; missing env vars produce warnings, not errors
- Obsidian config is optional; `pulse obsidian` tells you what to set if missing

## Testing Strategy

- **Table-driven tests for Collectors** — each adapter gets a test case row: mock input (HTTP response JSON, git command output, docker ps output) → expected DB rows. Same harness for all; adding an adapter = adding one test row.
- **Unit tests** for each Collector and Writer behind interfaces — mock the Store and Config, verify correct data is written/read
- **Unit tests** for the Briefing Engine — given known DB state, assert correct Briefing struct composition
- **Integration tests** for the SQLite Store — real DB (no mocks), test schema migrations, queries, and time-travel queries across sync runs
- **Integration tests** for `pulse sync --only` — with a fake Collector, verify end-to-end sync flow
- **CLI tests** — Cobra's built-in test helpers for command parsing, flag handling, and exit codes
- **TUI tests** — Bubble Tea `teatest` package for programmatic testing: tab switching, drill-down navigation, key bindings
- **Golden file tests** for stdout Writer — snapshot rendered briefing output, catch formatting regressions
- **Coverage target:** 100%

## Tech Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go | Single binary, fast, consistent with Nexus/Corvade/Vexade |
| CLI framework | Cobra | Standard for Go CLIs, used across other projects |
| TUI framework | Bubble Tea | Best Go TUI library, `teatest` for testing |
| Database | SQLite (modernc/sqlite) | Pure Go, no CGO, single file, used across other projects |
| SQL toolkit | sqlc | Type-safe queries from SQL, used in other projects |
| Config | Viper | YAML + env var support, pairs with Cobra |
| HTTP client | net/http | Standard library, no external deps needed |

## Distribution

- Single Go binary, installable via `go install` or Homebrew tap
- All config/data paths are absolute (`~/.config/pulse/`) — runs from any directory
- Cron setup documented; `pulse config init` can optionally install the cron entry
