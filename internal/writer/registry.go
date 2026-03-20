package writer

import (
	"context"
	"sync"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
)

// Writer renders a briefing to an output target.
type Writer interface {
	Name() string
	Write(ctx context.Context, briefing *domain.Briefing, cfg *config.Config) error
}

var (
	mu       sync.RWMutex
	registry = make(map[string]Writer)
)

// Register adds a writer to the global registry.
func Register(w Writer) {
	mu.Lock()
	defer mu.Unlock()
	registry[w.Name()] = w
}

// Get returns a writer by name.
func Get(name string) (Writer, bool) {
	mu.RLock()
	defer mu.RUnlock()
	w, ok := registry[name]
	return w, ok
}

// All returns all registered writers.
func All() []Writer {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Writer, 0, len(registry))
	for _, w := range registry {
		result = append(result, w)
	}
	return result
}

func init() {
	Register(NewStdoutWriter(nil))
	Register(NewObsidianWriter())
}
