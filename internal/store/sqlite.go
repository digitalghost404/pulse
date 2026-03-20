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
