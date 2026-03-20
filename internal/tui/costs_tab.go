package tui

import (
	"fmt"
	"strings"

	"github.com/xcoleman/pulse/internal/domain"
)

func renderCostsTab(cs domain.CostSummary, selected int, detail bool, width int) string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render(fmt.Sprintf("Costs (%s)", cs.Period)))
	sb.WriteString("\n\n")

	if cs.TotalCents == 0 {
		sb.WriteString(dimStyle.Render("  No cost data available"))
		return sb.String()
	}

	for i, sc := range cs.ByService {
		cursor := "  "
		if i == selected {
			cursor = "> "
		}

		// Bar chart
		barWidth := 20
		pct := float64(sc.AmountCents) / float64(cs.TotalCents)
		filled := int(pct * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		sb.WriteString(fmt.Sprintf("%s%-12s $%7.2f %s\n", cursor, sc.Service, float64(sc.AmountCents)/100, bar))
	}

	sb.WriteString(fmt.Sprintf("\n  Total: $%.2f — Burn: $%.2f/day\n",
		float64(cs.TotalCents)/100, float64(cs.BurnRateCents)/100))

	return sb.String()
}
