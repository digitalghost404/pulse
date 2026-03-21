package tui_test

import (
	"strings"
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
		Notifications: []domain.Notification{
			{RepoName: "obsidian-mcp", Type: "pr", Title: "Fix FTS5 indexing"},
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

func updateModel(m tui.Model, msg tea.Msg) tui.Model {
	model, _ := m.Update(msg)
	return model.(tui.Model)
}

func TestTabSwitching(t *testing.T) {
	m := tui.NewModel(sampleBriefing())
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Default is briefing tab — should contain project names
	view := m.View()
	if !strings.Contains(view, "cortex") {
		t.Error("briefing tab should contain 'cortex'")
	}
	if !strings.Contains(view, "pulse") {
		t.Error("briefing tab should contain 'pulse'")
	}

	// Switch to projects — should show project list
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	view = m.View()
	if !strings.Contains(view, "cortex") {
		t.Error("projects tab should contain 'cortex'")
	}
	if !strings.Contains(view, "Projects") {
		t.Error("projects tab should contain 'Projects' title")
	}

	// Switch to costs — should show service names and totals
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	view = m.View()
	if !strings.Contains(view, "claude") {
		t.Error("costs tab should contain 'claude'")
	}
	if !strings.Contains(view, "$18.42") {
		t.Error("costs tab should contain total '$18.42'")
	}
}

func TestProjectNavigation(t *testing.T) {
	m := tui.NewModel(sampleBriefing())
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Switch to projects
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})

	// First item should have cursor
	view := m.View()
	if !strings.Contains(view, "> ") {
		t.Error("expected cursor '> ' on first project")
	}

	// Move down
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	// Enter detail
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})

	// Esc to close
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})

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

func TestHelpToggle(t *testing.T) {
	m := tui.NewModel(sampleBriefing())
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Press ?
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	view := m.View()
	if !strings.Contains(view, "Key Bindings") {
		t.Error("expected help text after pressing ?")
	}

	// Press ? again to toggle off
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	view = m.View()
	if strings.Contains(view, "Key Bindings") {
		t.Error("expected help text to be hidden after toggling")
	}
}

func TestBriefingTabContent(t *testing.T) {
	m := tui.NewModel(sampleBriefing())
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	view := m.View()

	// Check all sections present
	for _, section := range []string{"Projects", "GitHub", "Costs", "System"} {
		if !strings.Contains(view, section) {
			t.Errorf("briefing tab should contain section %q", section)
		}
	}

	// Check notification
	if !strings.Contains(view, "Fix FTS5 indexing") {
		t.Error("briefing tab should contain notification title")
	}
}

func TestCostsNavigation(t *testing.T) {
	m := tui.NewModel(sampleBriefing())
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Switch to costs
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})

	// First service should have cursor
	view := m.View()
	if !strings.Contains(view, "> ") {
		t.Error("expected cursor on first cost service")
	}

	// Move down to voyage
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	view = m.View()
	if !strings.Contains(view, "voyage") {
		t.Error("expected voyage in costs view")
	}

	// Enter detail — should show extra info
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	view = m.View()
	if !strings.Contains(view, "Share:") {
		t.Error("expected drill-down detail with 'Share:' after pressing Enter")
	}
	if !strings.Contains(view, "Daily:") {
		t.Error("expected drill-down detail with 'Daily:' after pressing Enter")
	}

	// Esc to close detail
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	view = m.View()
	if strings.Contains(view, "Share:") {
		t.Error("expected detail to be closed after Esc")
	}
}

func TestCostsTabBarChart(t *testing.T) {
	m := tui.NewModel(sampleBriefing())
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Switch to costs
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	view := m.View()

	// Should contain bar chart characters
	if !strings.Contains(view, "█") {
		t.Error("costs tab should contain bar chart characters")
	}
}
