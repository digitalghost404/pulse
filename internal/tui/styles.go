package tui

import "github.com/charmbracelet/lipgloss"

var (
	tabStyle = lipgloss.NewStyle().
			Padding(0, 2)

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			MarginBottom(1)

	sectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8FBC8F")).
			Bold(true)

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F0C674"))

	okStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8FBC8F"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444"))
)
