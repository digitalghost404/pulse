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
