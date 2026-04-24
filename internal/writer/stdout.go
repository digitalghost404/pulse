// Package writer renders Pulse briefings to various output formats.
package writer

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
)

type StdoutWriter struct {
	out io.Writer
}

func NewStdoutWriter(out io.Writer) *StdoutWriter {
	if out == nil {
		out = os.Stdout
	}
	return &StdoutWriter{out: out}
}

func (w *StdoutWriter) Name() string { return "stdout" }

func (w *StdoutWriter) Write(ctx context.Context, b *domain.Briefing, cfg *config.Config) error {
	fmt.Fprintf(w.out, "Pulse Briefing — %s\n\n", b.GeneratedAt.Format("Mon Jan 2, 2006"))

	w.writeProjects(b.Projects)
	w.writeNotifications(b.Notifications)
	w.writeCosts(b.CostSummary)
	w.writeDocker(b.Docker)
	w.writeSystem(b.System)

	return nil
}

func (w *StdoutWriter) writeProjects(projects []domain.ProjectSummary) {
	fmt.Fprintf(w.out, "--- Projects ---\n")
	for _, p := range projects {
		icon := "✓"
		details := "clean"
		if p.DirtyFiles > 0 || p.Ahead > 0 || p.Behind > 0 {
			icon = "⚠"
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
		fmt.Fprintf(w.out, "  %s %s (%s) — %s\n", icon, p.RepoName, p.Branch, details)
	}
	fmt.Fprintln(w.out)
}

func (w *StdoutWriter) writeNotifications(notifs []domain.Notification) {
	if len(notifs) == 0 {
		return
	}
	fmt.Fprintf(w.out, "--- GitHub ---\n")
	for _, n := range notifs {
		icon := "●"
		fmt.Fprintf(w.out, "  %s %s — %s [%s]\n", icon, n.RepoName, n.Title, n.Type)
	}
	fmt.Fprintln(w.out)
}

func (w *StdoutWriter) writeCosts(cs domain.CostSummary) {
	if len(cs.ByService) == 0 {
		return
	}
	fmt.Fprintf(w.out, "--- Costs (%s) ---\n", cs.Period)
	for _, sc := range cs.ByService {
		if sc.AmountCents > 0 {
			fmt.Fprintf(w.out, "  %s: $%.2f", sc.Service, float64(sc.AmountCents)/100)
		} else {
			fmt.Fprintf(w.out, "  %s:", sc.Service)
		}
		if sc.UsageQuantity > 0 {
			fmt.Fprintf(w.out, " (%.0f %s)", sc.UsageQuantity, sc.UsageUnit)
		}
		fmt.Fprintln(w.out)
	}
	if cs.TotalCents > 0 {
		fmt.Fprintf(w.out, "  Total: $%.2f — Burn: $%.2f/day\n", float64(cs.TotalCents)/100, float64(cs.BurnRateCents)/100)
	}
	fmt.Fprintln(w.out)
}

func (w *StdoutWriter) writeDocker(containers []domain.DockerSnapshot) {
	if len(containers) == 0 {
		return
	}
	fmt.Fprintf(w.out, "--- Docker ---\n")
	for _, c := range containers {
		fmt.Fprintf(w.out, "  %s (%s) — %s\n", c.ContainerName, c.Image, c.Status)
	}
	fmt.Fprintln(w.out)
}

func (w *StdoutWriter) writeSystem(sys domain.SystemSnapshot) {
	fmt.Fprintf(w.out, "--- System ---\n")
	fmt.Fprintf(w.out, "  CPU: %.0f%% — RAM: %.1f/%.1f GB — Disk: %.0f/%.0f GB\n",
		sys.CPUPct,
		sys.MemoryUsedMB/1024, sys.MemoryTotalMB/1024,
		sys.DiskUsedGB, sys.DiskTotalGB)
	fmt.Fprintln(w.out)
}
