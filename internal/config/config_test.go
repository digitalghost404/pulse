package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
journal:
  watch_units:
    - ssh.service
    - docker.service
  min_priority: 4
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
	if len(cfg.Journal.WatchUnits) != 2 || cfg.Journal.WatchUnits[0] != "ssh.service" {
		t.Errorf("expected journal watch units loaded, got %v", cfg.Journal.WatchUnits)
	}
	if cfg.Journal.MinPriority != 4 {
		t.Errorf("expected journal min priority 4, got %d", cfg.Journal.MinPriority)
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

func TestObsidianDailyNotePath(t *testing.T) {
	cfg := &config.Config{
		Obsidian: config.ObsidianConfig{
			VaultPath:     "/vault",
			DailyNotePath: "Daily Notes/YYYY-MM-DD.md",
		},
	}

	testTime := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	result := cfg.ObsidianDailyNotePath(testTime)

	expected := filepath.Join("/vault", "Daily Notes/2026-03-20.md")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestObsidianDailyNotePath_CustomFormat(t *testing.T) {
	cfg := &config.Config{
		Obsidian: config.ObsidianConfig{
			VaultPath:     "/vault",
			DailyNotePath: "YYYY/MM/DD.md",
		},
	}

	testTime := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	result := cfg.ObsidianDailyNotePath(testTime)

	expected := filepath.Join("/vault", "2026/01/05.md")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
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
