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

		// Detail view for selected service
		if i == selected && detail {
			sb.WriteString("\n")
			sb.WriteString(fmt.Sprintf("      Service:  %s\n", sc.Service))
			sb.WriteString(fmt.Sprintf("      Amount:   $%.2f\n", float64(sc.AmountCents)/100))
			if sc.UsageQuantity > 0 {
				sb.WriteString(fmt.Sprintf("      Usage:    %.0f %s\n", sc.UsageQuantity, sc.UsageUnit))
			}
			if cs.TotalCents > 0 {
				sb.WriteString(fmt.Sprintf("      Share:    %.0f%%\n", pct*100))
			}
			// Use the overall burn rate to derive per-service daily rate
			var dailyRate float64
			if cs.TotalCents > 0 {
				dailyRate = float64(cs.BurnRateCents) * pct
			}
			sb.WriteString(fmt.Sprintf("      Daily:    ~$%.2f/day\n", dailyRate/100))
			sb.WriteString("\n")
		}
	}

	sb.WriteString(fmt.Sprintf("\n  Total: $%.2f — Burn: $%.2f/day\n",
		float64(cs.TotalCents)/100, float64(cs.BurnRateCents)/100))

	return sb.String()
}
