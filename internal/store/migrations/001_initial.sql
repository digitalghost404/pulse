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
