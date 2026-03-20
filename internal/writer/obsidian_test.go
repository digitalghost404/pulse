package writer_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/writer"
)

func TestObsidianWriter_CreatesSection(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "2026-03-20.md")

	// Create existing daily note
	existing := "# Daily Note\n\nSome content here.\n"
	os.WriteFile(notePath, []byte(existing), 0644)

	cfg := &config.Config{
		Obsidian: config.ObsidianConfig{
			VaultPath:      dir,
			DailyNotePath:  "YYYY-MM-DD.md",
			SectionHeading: "## Pulse Briefing",
		},
	}

	w := writer.NewObsidianWriter()
	b := sampleBriefing()

	err := w.Write(context.Background(), b, cfg)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	content, _ := os.ReadFile(notePath)
	if !strings.Contains(string(content), "## Pulse Briefing") {
		t.Error("expected briefing section heading")
	}
	if !strings.Contains(string(content), "cortex") {
		t.Error("expected project data in note")
	}
	if !strings.Contains(string(content), "Some content here.") {
		t.Error("expected existing content preserved")
	}
}

func TestObsidianWriter_MissingConfig(t *testing.T) {
	cfg := &config.Config{} // No obsidian config

	w := writer.NewObsidianWriter()
	err := w.Write(context.Background(), sampleBriefing(), cfg)

	if err == nil {
		t.Error("expected error for missing obsidian config")
	}
}
