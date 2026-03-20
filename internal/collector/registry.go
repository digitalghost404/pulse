package collector

import (
	"context"
	"fmt"
	"sync"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
)

// Collector gathers data from an external source and writes it to the store.
type Collector interface {
	Name() string
	EnvVars() []string
	Enabled(cfg *config.Config) bool
	Collect(ctx context.Context, s store.Store, cfg *config.Config, syncID int64) error
}

var (
	mu       sync.RWMutex
	registry = make(map[string]Collector)
)

// Register adds a collector to the global registry.
func Register(c Collector) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[c.Name()]; exists {
		panic(fmt.Sprintf("collector %q already registered", c.Name()))
	}
	registry[c.Name()] = c
}

// Get returns a collector by name.
func Get(name string) (Collector, bool) {
	mu.RLock()
	defer mu.RUnlock()
	c, ok := registry[name]
	return c, ok
}

// All returns all registered collectors.
func All() []Collector {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Collector, 0, len(registry))
	for _, c := range registry {
		result = append(result, c)
	}
	return result
}

// Enabled returns all collectors that are enabled in the config.
func Enabled(cfg *config.Config) []Collector {
	all := All()
	result := make([]Collector, 0, len(all))
	for _, c := range all {
		if c.Enabled(cfg) {
			result = append(result, c)
		}
	}
	return result
}
