package briefing

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

type Engine struct {
	store store.Store
	cfg   *config.Config
}

func NewEngine(s store.Store, cfg *config.Config) *Engine {
	return &Engine{store: s, cfg: cfg}
}

// BuildOptions configures briefing generation.
type BuildOptions struct {
	Since time.Time // override briefing time window
}

func (e *Engine) BuildWithOptions(ctx context.Context, opts BuildOptions) (*domain.Briefing, error) {
	// If no explicit since, use last briefing time
	if opts.Since.IsZero() {
		lastBriefing, _ := e.store.GetLastBriefingTime(ctx)
		if !lastBriefing.IsZero() {
			opts.Since = lastBriefing
		}
	}
	return e.build(ctx, opts.Since)
}

func (e *Engine) Build(ctx context.Context) (*domain.Briefing, error) {
	return e.BuildWithOptions(ctx, BuildOptions{})
}

func (e *Engine) build(ctx context.Context, since time.Time) (*domain.Briefing, error) {
	syncID, err := e.store.LatestSyncID(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting latest sync: %w", err)
	}
	if syncID == 0 {
		return nil, fmt.Errorf("no sync data available — run 'pulse sync' first")
	}

	b := &domain.Briefing{
		GeneratedAt: time.Now(),
	}

	// Projects
	snapshots, err := e.store.GetGitSnapshots(ctx, syncID)
	if err != nil {
		return nil, fmt.Errorf("reading git snapshots: %w", err)
	}
	for _, snap := range snapshots {
		branches, err := e.store.GetGitBranches(ctx, syncID, snap.RepoPath)
		if err != nil {
			log.Printf("WARN: reading branches for %s: %v", snap.RepoName, err)
		}
		b.Projects = append(b.Projects, domain.ProjectSummary{
			GitSnapshot: snap,
			Branches:    branches,
		})
	}

	// Notifications
	notifs, err := e.store.GetGitHubNotifications(ctx, syncID)
	if err != nil {
		log.Printf("WARN: reading GitHub notifications: %v", err)
	}
	b.Notifications = notifs

	// Costs — use since if provided, otherwise use config period
	costSince := since
	if costSince.IsZero() {
		costSince = ParsePeriod(e.cfg.Costs.DefaultPeriod)
	}
	costEntries, err := e.store.GetCostEntries(ctx, costSince)
	if err != nil {
		log.Printf("WARN: reading cost entries: %v", err)
	}
	b.CostSummary = buildCostSummary(costEntries, e.cfg.Costs.Currency, e.cfg.Costs.DefaultPeriod, costSince)

	// Docker
	docker, err := e.store.GetDockerSnapshots(ctx, syncID)
	if err != nil {
		log.Printf("WARN: reading Docker snapshots: %v", err)
	}
	b.Docker = docker

	// System
	sys, err := e.store.GetSystemSnapshot(ctx, syncID)
	if err != nil {
		log.Printf("WARN: reading system snapshot: %v", err)
	}
	if sys != nil {
		b.System = *sys
	}

	return b, nil
}

func buildCostSummary(entries []domain.CostEntry, currency, period string, since time.Time) domain.CostSummary {
	summary := domain.CostSummary{
		Currency: currency,
		Period:   period,
	}

	byService := make(map[string]*domain.ServiceCost)
	for _, e := range entries {
		sc, ok := byService[e.Service]
		if !ok {
			sc = &domain.ServiceCost{Service: e.Service, UsageUnit: e.UsageUnit}
			byService[e.Service] = sc
		}
		sc.AmountCents += e.AmountCents
		sc.UsageQuantity += e.UsageQuantity
		summary.TotalCents += e.AmountCents
	}

	// Sort services alphabetically for deterministic output
	for _, sc := range byService {
		summary.ByService = append(summary.ByService, *sc)
	}
	sort.Slice(summary.ByService, func(i, j int) bool {
		return summary.ByService[i].Service < summary.ByService[j].Service
	})

	// Burn rate: total cents / actual days of data (time from since to now)
	actualDays := time.Since(since).Hours() / 24
	if actualDays < 1 {
		actualDays = 1
	}
	if summary.TotalCents > 0 {
		summary.BurnRateCents = int(float64(summary.TotalCents) / actualDays)
	}

	return summary
}

// ParsePeriod converts a period string like "30d" to a time.Time in the past.
func ParsePeriod(period string) time.Time {
	period = strings.TrimSpace(period)
	if period == "" {
		period = "30d"
	}

	// Parse "Nd" format
	if strings.HasSuffix(period, "d") {
		numStr := strings.TrimSuffix(period, "d")
		var days int
		fmt.Sscanf(numStr, "%d", &days)
		if days > 0 {
			return time.Now().Add(-time.Duration(days) * 24 * time.Hour)
		}
	}

	// Default: 30 days
	return time.Now().Add(-30 * 24 * time.Hour)
}
