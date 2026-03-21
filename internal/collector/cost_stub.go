package collector

import (
	"context"
	"log"
	"os"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
)

// costStub is a placeholder for cost services without a usable billing API.
type costStub struct {
	name   string
	envVar string
	reason string
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
	log.Printf("WARN: %s cost collector is a stub — %s", c.name, c.reason)
	return nil
}

func init() {
	Register(&costStub{
		name:   "claude",
		envVar: "ANTHROPIC_API_KEY",
		reason: "usage API requires an Admin key (sk-ant-admin...), not a regular API key",
	})
	Register(&costStub{
		name:   "voyage",
		envVar: "VOYAGE_API_KEY",
		reason: "Voyage AI has no public billing/usage API — usage is only visible on the web dashboard",
	})
}
