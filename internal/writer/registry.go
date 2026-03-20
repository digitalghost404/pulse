package writer

import (
	"context"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
)

// Writer renders a briefing to an output target.
type Writer interface {
	Name() string
	Write(ctx context.Context, briefing *domain.Briefing, cfg *config.Config) error
}
