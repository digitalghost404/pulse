CREATE INDEX IF NOT EXISTS idx_cost_entries_period_end ON cost_entries(period_end);
CREATE INDEX IF NOT EXISTS idx_cost_entries_service ON cost_entries(service);
CREATE INDEX IF NOT EXISTS idx_sync_runs_status ON sync_runs(status);

INSERT INTO schema_version (version) VALUES (2);
