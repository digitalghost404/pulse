package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xcoleman/pulse/internal/domain"
)

type tab int

const (
	tabBriefing tab = iota
	tabProjects
	tabCosts
)

type Model struct {
	briefing     *domain.Briefing
	activeTab    tab
	width        int
	height       int
	projSelected int
	projDetail   bool
	costSelected int
	costDetail   bool
	showHelp     bool
}

func NewModel(b *domain.Briefing) Model {
	return Model{
		briefing:  b,
		activeTab: tabBriefing,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.activeTab = tabBriefing
		case "2":
			m.activeTab = tabProjects
			m.projDetail = false
		case "3":
			m.activeTab = tabCosts
			m.costDetail = false
		case "j", "down":
			m.moveDown()
		case "k", "up":
			m.moveUp()
		case "enter":
			m.toggleDetail()
		case "esc":
			m.closeDetail()
			m.showHelp = false
		case "?":
			m.showHelp = !m.showHelp
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *Model) moveDown() {
	switch m.activeTab {
	case tabProjects:
		if m.projSelected < len(m.briefing.Projects)-1 {
			m.projSelected++
		}
	case tabCosts:
		if m.costSelected < len(m.briefing.CostSummary.ByService)-1 {
			m.costSelected++
		}
	}
}

func (m *Model) moveUp() {
	switch m.activeTab {
	case tabProjects:
		if m.projSelected > 0 {
			m.projSelected--
		}
	case tabCosts:
		if m.costSelected > 0 {
			m.costSelected--
		}
	}
}

func (m *Model) toggleDetail() {
	switch m.activeTab {
	case tabProjects:
		m.projDetail = !m.projDetail
	case tabCosts:
		m.costDetail = !m.costDetail
	}
}

func (m *Model) closeDetail() {
	m.projDetail = false
	m.costDetail = false
}

func (m Model) View() string {
	var sb strings.Builder

	// Tab bar
	tabs := []struct {
		label string
		key   string
		t     tab
	}{
		{"Briefing", "1", tabBriefing},
		{"Projects", "2", tabProjects},
		{"Costs", "3", tabCosts},
	}

	var tabParts []string
	for _, t := range tabs {
		label := fmt.Sprintf("%s %s", t.key, t.label)
		if t.t == m.activeTab {
			tabParts = append(tabParts, activeTabStyle.Render(label))
		} else {
			tabParts = append(tabParts, tabStyle.Render(label))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabParts...)
	sb.WriteString(tabBar)
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", m.width))
	sb.WriteString("\n\n")

	// Content
	switch m.activeTab {
	case tabBriefing:
		sb.WriteString(renderBriefingTab(m.briefing, m.width))
	case tabProjects:
		sb.WriteString(renderProjectsTab(m.briefing.Projects, m.projSelected, m.projDetail, m.width))
	case tabCosts:
		sb.WriteString(renderCostsTab(m.briefing.CostSummary, m.costSelected, m.costDetail, m.width))
	}

	// Help
	sb.WriteString("\n")
	if m.showHelp {
		sb.WriteString(helpStyle.Render(
			"Key Bindings:\n" +
				"  1        Briefing tab\n" +
				"  2        Projects tab\n" +
				"  3        Costs tab\n" +
				"  j/k      Scroll down/up\n" +
				"  Enter    Drill into selected item\n" +
				"  Esc      Back / close help\n" +
				"  ?        Toggle this help\n" +
				"  q        Quit\n"))
	} else {
		sb.WriteString(helpStyle.Render("q quit · 1-3 tabs · j/k scroll · enter drill · esc back · ? help"))
	}

	return sb.String()
}
