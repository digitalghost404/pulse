package tui

import (
	"fmt"
	"strings"

	"github.com/xcoleman/pulse/internal/domain"
)

func renderBriefingTab(b *domain.Briefing, width int) string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render(fmt.Sprintf("Pulse Briefing — %s", b.GeneratedAt.Format("Mon Jan 2, 2006"))))
	sb.WriteString("\n\n")

	// Projects
	sb.WriteString(sectionStyle.Render("--- Projects ---"))
	sb.WriteString("\n")
	for _, p := range b.Projects {
		icon := okStyle.Render("✓")
		details := "clean"
		if p.DirtyFiles > 0 || p.Ahead > 0 || p.Behind > 0 {
			icon = warnStyle.Render("⚠")
			var parts []string
			if p.DirtyFiles > 0 {
				parts = append(parts, fmt.Sprintf("%d dirty", p.DirtyFiles))
			}
			if p.Ahead > 0 {
				parts = append(parts, fmt.Sprintf("%d ahead", p.Ahead))
			}
			if p.Behind > 0 {
				parts = append(parts, fmt.Sprintf("%d behind", p.Behind))
			}
			details = strings.Join(parts, ", ")
		}
		fmt.Fprintf(&sb, "  %s %s (%s) — %s\n", icon, p.RepoName, p.Branch, details)
	}

	sb.WriteString("\n")

	// Notifications
	if len(b.Notifications) > 0 {
		sb.WriteString(sectionStyle.Render("--- GitHub ---"))
		sb.WriteString("\n")
		for _, n := range b.Notifications {
			fmt.Fprintf(&sb, "  ● %s — %s [%s]\n", n.RepoName, n.Title, n.Type)
		}
		sb.WriteString("\n")
	}

	// Costs
	if b.CostSummary.TotalCents > 0 {
		sb.WriteString(sectionStyle.Render(fmt.Sprintf("--- Costs (%s) ---", b.CostSummary.Period)))
		sb.WriteString("\n")
		for _, sc := range b.CostSummary.ByService {
			fmt.Fprintf(&sb, "  %s: $%.2f\n", sc.Service, float64(sc.AmountCents)/100)
		}
		fmt.Fprintf(&sb, "  Total: $%.2f — Burn: $%.2f/day\n",
			float64(b.CostSummary.TotalCents)/100, float64(b.CostSummary.BurnRateCents)/100)
		sb.WriteString("\n")
	}

	// Docker
	if len(b.Docker) > 0 {
		sb.WriteString(sectionStyle.Render("--- Docker ---"))
		sb.WriteString("\n")
		for _, c := range b.Docker {
			icon := okStyle.Render("●")
			if !strings.HasPrefix(c.Status, "Up") {
				icon = dimStyle.Render("○")
			}
			fmt.Fprintf(&sb, "  %s %s (%s) — %s\n", icon, c.ContainerName, c.Image, c.Status)
		}
		sb.WriteString("\n")
	}

	// System
	sb.WriteString(sectionStyle.Render("--- System ---"))
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "  CPU: %.0f%% — RAM: %.1f/%.1f GB — Disk: %.0f/%.0f GB\n",
		b.System.CPUPct,
		b.System.MemoryUsedMB/1024, b.System.MemoryTotalMB/1024,
		b.System.DiskUsedGB, b.System.DiskTotalGB)

	return sb.String()
}
