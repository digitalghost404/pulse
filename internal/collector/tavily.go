package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

type TavilyCollector struct{}

func (t *TavilyCollector) Name() string      { return "tavily" }
func (t *TavilyCollector) EnvVars() []string { return []string{"TAVILY_API_KEY"} }

func (t *TavilyCollector) Enabled(cfg *config.Config) bool {
	if !cfg.AdapterEnabled("tavily") {
		return false
	}
	return os.Getenv("TAVILY_API_KEY") != ""
}

type tavilyUsageResponse struct {
	Key struct {
		Usage         int `json:"usage"`
		Limit         int `json:"limit"`
		SearchUsage   int `json:"search_usage"`
		ExtractUsage  int `json:"extract_usage"`
		CrawlUsage    int `json:"crawl_usage"`
		MapUsage      int `json:"map_usage"`
		ResearchUsage int `json:"research_usage"`
	} `json:"key"`
	Account struct {
		CurrentPlan string `json:"current_plan"`
		PlanUsage   int    `json:"plan_usage"`
		PlanLimit   int    `json:"plan_limit"`
		PaygoUsage  int    `json:"paygo_usage"`
		PaygoLimit  int    `json:"paygo_limit"`
	} `json:"account"`
}

type tavilyRawData struct {
	CumulativeUsage int `json:"cumulative_usage"`
	DeltaUsage      int `json:"delta_usage"`
}

func (t *TavilyCollector) Collect(ctx context.Context, s store.Store, cfg *config.Config, syncID int64) error {
	apiKey := os.Getenv("TAVILY_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("TAVILY_API_KEY not set")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.tavily.com/usage", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching Tavily usage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tavily API returned %d (check TAVILY_API_KEY)", resp.StatusCode)
	}

	body, err := limitedReadAll(resp.Body)
	if err != nil {
		return err
	}

	var usage tavilyUsageResponse
	if err := json.Unmarshal(body, &usage); err != nil {
		return fmt.Errorf("parsing Tavily usage: %w", err)
	}

	// Compute delta from last known cumulative usage
	currentCumulative := usage.Key.Usage
	deltaUsage := currentCumulative // default: treat full amount as delta (first run)

	prev, err := s.GetLatestCostEntry(ctx, "tavily")
	if err == nil && prev != nil {
		var prevRaw tavilyRawData
		if json.Unmarshal([]byte(prev.RawData), &prevRaw) == nil && prevRaw.CumulativeUsage > 0 {
			delta := currentCumulative - prevRaw.CumulativeUsage
			if delta >= 0 {
				deltaUsage = delta
			}
			// If delta < 0, usage counter was reset (new billing cycle) — use full amount
		}
	}

	now := time.Now()
	amountCents := 0
	if cfg.Costs.Pricing.TavilyCentsPerRequest > 0 {
		amountCents = deltaUsage * cfg.Costs.Pricing.TavilyCentsPerRequest
	}

	rawData, _ := json.Marshal(tavilyRawData{
		CumulativeUsage: currentCumulative,
		DeltaUsage:      deltaUsage,
	})

	entry := domain.CostEntry{
		Service:       "tavily",
		PeriodStart:   now.Add(-24 * time.Hour),
		PeriodEnd:     now,
		AmountCents:   amountCents,
		Currency:      "USD",
		UsageQuantity: float64(deltaUsage),
		UsageUnit:     "requests",
		RawData:       string(rawData),
	}

	return s.SaveCostEntry(ctx, syncID, entry)
}

func init() {
	Register(&TavilyCollector{})
}
