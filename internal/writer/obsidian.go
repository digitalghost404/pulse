package writer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
)

type ObsidianWriter struct{}

func NewObsidianWriter() *ObsidianWriter {
	return &ObsidianWriter{}
}

func (w *ObsidianWriter) Name() string { return "obsidian" }

func (w *ObsidianWriter) Write(ctx context.Context, b *domain.Briefing, cfg *config.Config) error {
	if cfg.Obsidian.VaultPath == "" {
		return fmt.Errorf("obsidian vault_path not configured — set it in ~/.config/pulse/config.yaml")
	}

	notePath := cfg.ObsidianDailyNotePath(b.GeneratedAt)
	heading := cfg.Obsidian.SectionHeading
	if heading == "" {
		heading = "## Pulse Briefing"
	}

	// Render briefing as markdown
	md := renderMarkdown(b)
	section := fmt.Sprintf("\n%s\n\n%s\n", heading, md)

	// Read existing note or create new
	existing, err := os.ReadFile(notePath)
	if err != nil {
		dir := filepath.Dir(notePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating note directory: %w", err)
		}
		return os.WriteFile(notePath, []byte(section), 0644)
	}

	// Check if section already exists — replace it
	content := string(existing)
	if idx := strings.Index(content, heading); idx >= 0 {
		rest := content[idx+len(heading):]
		nextHeading := strings.Index(rest, "\n## ")
		if nextHeading >= 0 {
			content = content[:idx] + section + rest[nextHeading:]
		} else {
			content = content[:idx] + section
		}
	} else {
		content = content + section
	}

	return os.WriteFile(notePath, []byte(content), 0644)
}

func renderMarkdown(b *domain.Briefing) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*Generated %s*\n\n", b.GeneratedAt.Format("Mon Jan 2, 2006 15:04")))

	// Projects
	sb.WriteString("### Projects\n\n")
	for _, p := range b.Projects {
		icon := "✅"
		details := "clean"
		if p.DirtyFiles > 0 || p.Ahead > 0 || p.Behind > 0 {
			icon = "⚠️"
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
		sb.WriteString(fmt.Sprintf("- %s **%s** (`%s`) — %s\n", icon, p.RepoName, p.Branch, details))
	}
	sb.WriteString("\n")

	// Notifications
	if len(b.Notifications) > 0 {
		sb.WriteString("### GitHub\n\n")
		for _, n := range b.Notifications {
			sb.WriteString(fmt.Sprintf("- 🔔 **%s** — %s `[%s]`\n", n.RepoName, n.Title, n.Type))
		}
		sb.WriteString("\n")
	}

	// Costs
	if b.CostSummary.TotalCents > 0 {
		sb.WriteString(fmt.Sprintf("### Costs (%s)\n\n", b.CostSummary.Period))
		sb.WriteString("| Service | Cost |\n|---------|------|\n")
		for _, sc := range b.CostSummary.ByService {
			sb.WriteString(fmt.Sprintf("| %s | $%.2f |\n", sc.Service, float64(sc.AmountCents)/100))
		}
		sb.WriteString(fmt.Sprintf("\n**Total:** $%.2f — **Burn:** $%.2f/day\n\n",
			float64(b.CostSummary.TotalCents)/100, float64(b.CostSummary.BurnRateCents)/100))
	}

	// Docker
	if len(b.Docker) > 0 {
		sb.WriteString("### Docker\n\n")
		for _, c := range b.Docker {
			sb.WriteString(fmt.Sprintf("- `%s` (%s) — %s\n", c.ContainerName, c.Image, c.Status))
		}
		sb.WriteString("\n")
	}

	// System
	sb.WriteString("### System\n\n")
	sb.WriteString(fmt.Sprintf("- **CPU:** %.0f%%\n- **RAM:** %.1f/%.1f GB\n- **Disk:** %.0f/%.0f GB\n",
		b.System.CPUPct,
		b.System.MemoryUsedMB/1024, b.System.MemoryTotalMB/1024,
		b.System.DiskUsedGB, b.System.DiskTotalGB))

	return sb.String()
}
