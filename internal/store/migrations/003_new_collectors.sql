CREATE TABLE IF NOT EXISTS hardware_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id INTEGER NOT NULL REFERENCES sync_runs(id),
    gpu_name TEXT,
    gpu_util_pct REAL,
    gpu_mem_used_mb REAL,
    gpu_mem_total_mb REAL,
    gpu_temp_c INTEGER,
    gpu_power_watts REAL,
    gpu_fan_speed_pct INTEGER,
    cpu_temp_c INTEGER,
    cpu_freq_mhz INTEGER,
    cpu_throttled BOOLEAN,
    battery_pct INTEGER,
    battery_status TEXT,
    battery_watts REAL,
    package_power_watts REAL,
    dram_power_watts REAL
);

CREATE TABLE IF NOT EXISTS network_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id INTEGER NOT NULL REFERENCES sync_runs(id),
    active_interface TEXT,
    connection_type TEXT,
    vpn_active BOOLEAN,
    vpn_provider TEXT,
    wifi_ssid TEXT,
    wifi_signal_dbm INTEGER,
    wifi_band TEXT,
    interfaces TEXT  -- JSON array of InterfaceStats
);

CREATE TABLE IF NOT EXISTS storage_health (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id INTEGER NOT NULL REFERENCES sync_runs(id),
    drives TEXT  -- JSON array of DriveHealth
);

CREATE TABLE IF NOT EXISTS journal_alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id INTEGER NOT NULL REFERENCES sync_runs(id),
    timestamp DATETIME,
    unit TEXT,
    priority INTEGER,
    message TEXT,
    category TEXT
);

CREATE INDEX IF NOT EXISTS idx_journal_alerts_sync ON journal_alerts(sync_id);
CREATE INDEX IF NOT EXISTS idx_hardware_snapshots_sync ON hardware_snapshots(sync_id);
CREATE INDEX IF NOT EXISTS idx_network_snapshots_sync ON network_snapshots(sync_id);
CREATE INDEX IF NOT EXISTS idx_storage_health_sync ON storage_health(sync_id);

INSERT INTO schema_version (version) VALUES (3);
