package collector

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
)

// costStub is a placeholder collector for cost services whose billing APIs
// need to be verified before full implementation.
type costStub struct {
	name   string
	envVar string
}

func (c *costStub) Name() string      { return c.name }
func (c *costStub) EnvVars() []string { return []string{c.envVar} }

func (c *costStub) Enabled(cfg *config.Config) bool {
	if !cfg.AdapterEnabled(c.name) {
		return false
	}
	return os.Getenv(c.envVar) != ""
}

func (c *costStub) Collect(ctx context.Context, s store.Store, cfg *config.Config, syncID int64) error {
	log.Printf("WARN: %s cost collector is a stub — billing API integration pending", c.name)
	return fmt.Errorf("%s cost collector not yet implemented", c.name)
}

func init() {
	Register(&costStub{name: "claude", envVar: "ANTHROPIC_API_KEY"})
	Register(&costStub{name: "voyage", envVar: "VOYAGE_API_KEY"})
	Register(&costStub{name: "tavily", envVar: "TAVILY_API_KEY"})
	Register(&costStub{name: "elevenlabs", envVar: "ELEVENLABS_API_KEY"})
}
