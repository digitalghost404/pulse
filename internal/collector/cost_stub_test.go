package collector_test

import (
	"os"
	"testing"

	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
)

func TestCostCollectors_Registered(t *testing.T) {
	for _, name := range []string{"claude", "voyage", "tavily", "elevenlabs"} {
		_, ok := collector.Get(name)
		if !ok {
			t.Errorf("expected cost collector %q to be registered", name)
		}
	}
}

func TestCostCollectors_EnvVarStubs(t *testing.T) {
	// Only voyage is still a stub with env var requirements
	tests := []struct {
		name   string
		envVar string
	}{
		{"voyage", "VOYAGE_API_KEY"},
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

func TestCostCollectors_RealCollectorEnvVars(t *testing.T) {
	// Tavily and ElevenLabs have real collectors with env vars
	tests := []struct {
		name   string
		envVar string
	}{
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

func TestClaudeCollector_NoEnvVarRequired(t *testing.T) {
	c, ok := collector.Get("claude")
	if !ok {
		t.Fatal("expected claude collector to be registered")
	}
	// Claude collector reads local logs — no API key needed
	if len(c.EnvVars()) != 0 {
		t.Errorf("expected no env vars for claude, got %v", c.EnvVars())
	}
	// Should be enabled by default (reads local files)
	cfg := &config.Config{}
	if !c.Enabled(cfg) {
		t.Error("expected claude collector enabled by default")
	}
}

func TestCostStub_DisabledWithoutEnvVar(t *testing.T) {
	cfg := &config.Config{}

	// Only voyage still requires env var
	c, _ := collector.Get("voyage")
	os.Unsetenv("VOYAGE_API_KEY")
	if c.Enabled(cfg) {
		t.Error("expected voyage collector disabled without env var")
	}
}
