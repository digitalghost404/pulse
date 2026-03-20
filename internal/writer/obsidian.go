package writer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
)

type ObsidianWriter struct{}

func NewObsidianWriter() *ObsidianWriter {
	return &ObsidianWriter{}
}

func (w *ObsidianWriter) Name() string { return "obsidian" }

func (w *ObsidianWriter) Write(ctx context.Context, b *domain.Briefing, cfg *config.Config) error {
	if cfg.Obsidian.VaultPath == "" {
		return fmt.Errorf("obsidian vault_path not configured — set it in ~/.config/pulse/config.yaml")
	}

	notePath := cfg.ObsidianDailyNotePath(b.GeneratedAt)
	heading := cfg.Obsidian.SectionHeading
	if heading == "" {
		heading = "## Pulse Briefing"
	}

	// Render briefing as markdown
	var md bytes.Buffer
	stdoutWriter := NewStdoutWriter(&md)
	stdoutWriter.Write(ctx, b, cfg)

	section := fmt.Sprintf("\n%s\n\n%s\n", heading, md.String())

	// Read existing note or create new
	existing, err := os.ReadFile(notePath)
	if err != nil {
		dir := filepath.Dir(notePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating note directory: %w", err)
		}
		return os.WriteFile(notePath, []byte(section), 0644)
	}

	// Check if section already exists — replace it
	content := string(existing)
	if idx := strings.Index(content, heading); idx >= 0 {
		rest := content[idx+len(heading):]
		nextHeading := strings.Index(rest, "\n## ")
		if nextHeading >= 0 {
			content = content[:idx] + section + rest[nextHeading:]
		} else {
			content = content[:idx] + section
		}
	} else {
		content = content + section
	}

	return os.WriteFile(notePath, []byte(content), 0644)
}
