package tui_test

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/tui"
)

func sampleBriefing() *domain.Briefing {
	return &domain.Briefing{
		GeneratedAt: time.Date(2026, 3, 20, 7, 0, 0, 0, time.Local),
		Projects: []domain.ProjectSummary{
			{GitSnapshot: domain.GitSnapshot{RepoName: "cortex", Branch: "main", DirtyFiles: 3, Ahead: 2}},
			{GitSnapshot: domain.GitSnapshot{RepoName: "pulse", Branch: "main"}},
		},
		CostSummary: domain.CostSummary{
			TotalCents: 1842, Currency: "USD", Period: "30d", BurnRateCents: 61,
			ByService: []domain.ServiceCost{
				{Service: "claude", AmountCents: 1482},
				{Service: "voyage", AmountCents: 210},
			},
		},
		System: domain.SystemSnapshot{CPUPct: 12.5, MemoryUsedMB: 18200, MemoryTotalMB: 32000},
	}
}

func TestTabSwitching(t *testing.T) {
	m := tui.NewModel(sampleBriefing())
	var cmd tea.Cmd
	var model tea.Model
	model, cmd = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = model.(tui.Model)
	_ = cmd

	// Default is briefing tab
	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}

	// Switch to projects
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = model.(tui.Model)
	view = m.View()
	if view == "" {
		t.Fatal("expected non-empty projects view")
	}

	// Switch to costs
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m = model.(tui.Model)
	view = m.View()
	if view == "" {
		t.Fatal("expected non-empty costs view")
	}
}

func TestProjectNavigation(t *testing.T) {
	m := tui.NewModel(sampleBriefing())
	var model tea.Model

	model, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = model.(tui.Model)

	// Switch to projects
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = model.(tui.Model)

	// Move down
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = model.(tui.Model)

	// Enter detail
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(tui.Model)

	// Esc to close
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = model.(tui.Model)

	// Should not panic
	_ = m.View()
}

func TestQuit(t *testing.T) {
	m := tui.NewModel(sampleBriefing())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if cmd == nil {
		t.Error("expected quit command")
	}
}
