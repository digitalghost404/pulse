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
	s := string(content)
	if !strings.Contains(s, "## Pulse Briefing") {
		t.Error("expected briefing section heading")
	}
	if !strings.Contains(s, "cortex") {
		t.Error("expected project data in note")
	}
	if !strings.Contains(s, "Some content here.") {
		t.Error("expected existing content preserved")
	}
}

func TestObsidianWriter_ReplacesExistingSection(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "2026-03-20.md")

	// Create note with existing pulse section
	existing := "# Daily Note\n\nSome content.\n\n## Pulse Briefing\n\nOld briefing data.\n\n## Other Section\n\nOther content.\n"
	os.WriteFile(notePath, []byte(existing), 0644)

	cfg := &config.Config{
		Obsidian: config.ObsidianConfig{
			VaultPath:      dir,
			DailyNotePath:  "YYYY-MM-DD.md",
			SectionHeading: "## Pulse Briefing",
		},
	}

	w := writer.NewObsidianWriter()
	err := w.Write(context.Background(), sampleBriefing(), cfg)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	content, _ := os.ReadFile(notePath)
	s := string(content)

	// Should have new data
	if !strings.Contains(s, "cortex") {
		t.Error("expected new project data")
	}
	// Old data should be gone
	if strings.Contains(s, "Old briefing data") {
		t.Error("expected old briefing data to be replaced")
	}
	// Other section should be preserved
	if !strings.Contains(s, "## Other Section") {
		t.Error("expected other section to be preserved")
	}
	if !strings.Contains(s, "Other content.") {
		t.Error("expected other section content preserved")
	}
}

func TestObsidianWriter_RendersMarkdown(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "2026-03-20.md")

	cfg := &config.Config{
		Obsidian: config.ObsidianConfig{
			VaultPath:      dir,
			DailyNotePath:  "YYYY-MM-DD.md",
			SectionHeading: "## Pulse Briefing",
		},
	}

	w := writer.NewObsidianWriter()
	err := w.Write(context.Background(), sampleBriefing(), cfg)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	content, _ := os.ReadFile(notePath)
	s := string(content)

	// Should contain markdown formatting
	if !strings.Contains(s, "### Projects") {
		t.Error("expected markdown heading '### Projects'")
	}
	if !strings.Contains(s, "### System") {
		t.Error("expected markdown heading '### System'")
	}
	if !strings.Contains(s, "**cortex**") {
		t.Error("expected bold markdown for project name")
	}
	if !strings.Contains(s, "| Service |") {
		t.Error("expected markdown table for costs")
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
