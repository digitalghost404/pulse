package collector_test

import (
	"testing"

	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
)

func TestCollectorRegistry_All(t *testing.T) {
	all := collector.All()
	// Should have at least: git, system, docker, github, claude, voyage, tavily, elevenlabs
	if len(all) < 8 {
		t.Errorf("expected at least 8 collectors, got %d", len(all))
	}
}

func TestCollectorRegistry_Get(t *testing.T) {
	for _, name := range []string{"git", "system", "docker", "github"} {
		c, ok := collector.Get(name)
		if !ok {
			t.Errorf("expected collector %q to be registered", name)
		}
		if c.Name() != name {
			t.Errorf("expected Name() = %q, got %q", name, c.Name())
		}
	}
}

func TestCollectorRegistry_GetNotFound(t *testing.T) {
	_, ok := collector.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent collector to not be found")
	}
}

func TestCollectorRegistry_Enabled(t *testing.T) {
	cfg := &config.Config{
		Adapters: map[string]bool{
			"git":    true,
			"docker": false,
			"system": true,
		},
	}

	enabled := collector.Enabled(cfg)

	names := make(map[string]bool)
	for _, c := range enabled {
		names[c.Name()] = true
	}

	if !names["git"] {
		t.Error("expected git in enabled collectors")
	}
	if names["docker"] {
		t.Error("expected docker NOT in enabled collectors")
	}
	if !names["system"] {
		t.Error("expected system in enabled collectors (explicitly enabled)")
	}
}
