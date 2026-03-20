package writer_test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/writer"
)

func sampleBriefing() *domain.Briefing {
	return &domain.Briefing{
		GeneratedAt: time.Date(2026, 3, 20, 7, 0, 0, 0, time.Local),
		Projects: []domain.ProjectSummary{
			{GitSnapshot: domain.GitSnapshot{RepoName: "cortex", Branch: "main", DirtyFiles: 3, Ahead: 2}},
			{GitSnapshot: domain.GitSnapshot{RepoName: "pulse", Branch: "main", DirtyFiles: 0, Ahead: 0}},
		},
		Notifications: []domain.Notification{
			{RepoName: "obsidian-mcp", Type: "pr", Title: "Fix FTS5 indexing", State: "open"},
		},
		CostSummary: domain.CostSummary{
			TotalCents: 1842, Currency: "USD", Period: "30d", BurnRateCents: 61,
			ByService: []domain.ServiceCost{
				{Service: "claude", AmountCents: 1482},
				{Service: "voyage", AmountCents: 210},
				{Service: "tavily", AmountCents: 150},
			},
		},
		System: domain.SystemSnapshot{
			CPUPct: 12.5, MemoryUsedMB: 18200, MemoryTotalMB: 32000,
			DiskUsedGB: 142, DiskTotalGB: 256,
		},
	}
}

func TestStdoutWriter_ContainsSections(t *testing.T) {
	var buf bytes.Buffer
	w := writer.NewStdoutWriter(&buf)
	cfg := &config.Config{}

	err := w.Write(context.Background(), sampleBriefing(), cfg)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	output := buf.String()

	// Check key sections are present
	sections := []string{"Projects", "GitHub", "Costs", "System"}
	for _, section := range sections {
		if !bytes.Contains([]byte(output), []byte(section)) {
			t.Errorf("expected output to contain section %q", section)
		}
	}

	// Check project data
	if !bytes.Contains([]byte(output), []byte("cortex")) {
		t.Error("expected output to contain 'cortex'")
	}
	if !bytes.Contains([]byte(output), []byte("pulse")) {
		t.Error("expected output to contain 'pulse'")
	}
}

func TestStdoutWriter_GoldenFile(t *testing.T) {
	var buf bytes.Buffer
	w := writer.NewStdoutWriter(&buf)
	cfg := &config.Config{}

	err := w.Write(context.Background(), sampleBriefing(), cfg)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	output := buf.Bytes()
	goldenPath := "testdata/briefing_full.golden"

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		os.WriteFile(goldenPath, output, 0644)
		t.Log("Golden file updated")
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("Golden file not found. Run with UPDATE_GOLDEN=1 to create: %v", err)
	}

	if !bytes.Equal(output, expected) {
		t.Errorf("stdout output differs from golden file. Run with UPDATE_GOLDEN=1 to update.\nGot:\n%s\nExpected:\n%s", output, expected)
	}
}

func TestStdoutWriter_CostFormatting(t *testing.T) {
	var buf bytes.Buffer
	w := writer.NewStdoutWriter(&buf)
	cfg := &config.Config{}

	err := w.Write(context.Background(), sampleBriefing(), cfg)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("$14.82")) {
		t.Error("expected output to contain '$14.82' for claude costs")
	}
	if !bytes.Contains([]byte(output), []byte("$18.42")) {
		t.Error("expected output to contain '$18.42' for total")
	}
}
