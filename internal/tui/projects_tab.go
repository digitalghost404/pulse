package tui

import (
	"fmt"
	"strings"

	"github.com/xcoleman/pulse/internal/domain"
)

func renderProjectsTab(projects []domain.ProjectSummary, selected int, detail bool, width int) string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Projects"))
	sb.WriteString("\n\n")

	for i, p := range projects {
		icon := okStyle.Render("✓")
		if p.DirtyFiles > 0 || p.Ahead > 0 || p.Behind > 0 {
			icon = warnStyle.Render("⚠")
		}

		cursor := "  "
		if i == selected {
			cursor = "> "
		}

		sb.WriteString(fmt.Sprintf("%s%s %s (%s)", cursor, icon, p.RepoName, p.Branch))

		if p.DirtyFiles > 0 {
			sb.WriteString(fmt.Sprintf(" — %d dirty", p.DirtyFiles))
		}
		if p.Ahead > 0 {
			sb.WriteString(fmt.Sprintf(", %d ahead", p.Ahead))
		}
		sb.WriteString("\n")

		// Show detail if selected and in detail mode
		if i == selected && detail {
			sb.WriteString(fmt.Sprintf("      Last commit: %s — %s\n", p.LastCommitHash, p.LastCommitMsg))
			if len(p.Branches) > 0 {
				sb.WriteString("      Branches:\n")
				for _, br := range p.Branches {
					merged := ""
					if br.IsMerged {
						merged = " [merged]"
					}
					current := " "
					if br.IsCurrent {
						current = "*"
					}
					sb.WriteString(fmt.Sprintf("        %s %s%s\n", current, br.BranchName, merged))
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
