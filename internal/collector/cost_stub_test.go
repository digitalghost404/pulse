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
