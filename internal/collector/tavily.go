package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching Tavily usage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Tavily API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var usage tavilyUsageResponse
	if err := json.Unmarshal(body, &usage); err != nil {
		return fmt.Errorf("parsing Tavily usage: %w", err)
	}

	now := time.Now()
	entry := domain.CostEntry{
		Service:       "tavily",
		PeriodStart:   now.Add(-24 * time.Hour),
		PeriodEnd:     now,
		AmountCents:   0, // Tavily doesn't expose dollar amounts via API
		Currency:      "USD",
		UsageQuantity: float64(usage.Key.Usage),
		UsageUnit:     "requests",
		RawData:       string(body),
	}

	return s.SaveCostEntry(ctx, syncID, entry)
}

func init() {
	Register(&TavilyCollector{})
}
