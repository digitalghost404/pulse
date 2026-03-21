package collector

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

type ClaudeCollector struct{}

func (c *ClaudeCollector) Name() string      { return "claude" }
func (c *ClaudeCollector) EnvVars() []string { return nil } // No API key needed — reads local logs

func (c *ClaudeCollector) Enabled(cfg *config.Config) bool {
	return cfg.AdapterEnabled("claude")
}

// claudeLogEntry represents a line from Claude Code's JSONL log.
type claudeLogEntry struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Message   struct {
		Model   string `json:"model"`
		Role    string `json:"role"`
		Usage   *claudeUsage `json:"usage"`
	} `json:"message"`
}

type claudeUsage struct {
	InputTokens              int    `json:"input_tokens"`
	OutputTokens             int    `json:"output_tokens"`
	CacheCreationInputTokens int    `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int    `json:"cache_read_input_tokens"`
	ServiceTier              string `json:"service_tier"`
}

// Model pricing per million tokens (as of 2025)
var modelPricing = map[string]struct{ input, output float64 }{
	"claude-opus-4-6":    {15.00, 75.00},
	"claude-opus-4-20250514":    {15.00, 75.00},
	"claude-sonnet-4-6":  {3.00, 15.00},
	"claude-sonnet-4-20250514":  {3.00, 15.00},
	"claude-3-5-haiku-20251001": {0.80, 4.00},
	"claude-haiku-4-5-20251001": {0.80, 4.00},
	// Older models
	"claude-3-5-sonnet-20241022": {3.00, 15.00},
	"claude-3-5-sonnet-20240620": {3.00, 15.00},
	"claude-3-opus-20240229":     {15.00, 75.00},
	"claude-3-haiku-20240307":    {0.25, 1.25},
}

func (c *ClaudeCollector) Collect(ctx context.Context, s store.Store, cfg *config.Config, syncID int64) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	claudeDir := filepath.Join(home, ".claude", "projects")
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		log.Printf("INFO: no Claude Code logs found at %s", claudeDir)
		return nil
	}

	// Scan all project dirs for JSONL files
	now := time.Now()
	since := now.Add(-24 * time.Hour)

	var totalInputTokens, totalOutputTokens, totalCacheCreateTokens, totalCacheReadTokens int
	modelUsage := make(map[string]*struct{ input, output int })

	err = filepath.Walk(claudeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		// Skip files not modified in the last 24 hours
		if info.ModTime().Before(since) {
			return nil
		}

		entries := parseJSONLFile(path, since)
		for _, e := range entries {
			totalInputTokens += e.inputTokens
			totalOutputTokens += e.outputTokens
			totalCacheCreateTokens += e.cacheCreate
			totalCacheReadTokens += e.cacheRead

			if e.model != "" {
				mu, ok := modelUsage[e.model]
				if !ok {
					mu = &struct{ input, output int }{}
					modelUsage[e.model] = mu
				}
				mu.input += e.inputTokens + e.cacheCreate + e.cacheRead
				mu.output += e.outputTokens
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("walking Claude logs: %w", err)
	}

	totalTokens := totalInputTokens + totalOutputTokens + totalCacheCreateTokens + totalCacheReadTokens
	if totalTokens == 0 {
		return nil // no usage in period
	}

	// Calculate cost
	var totalCostCents int
	usageUnit := "tokens"
	subscription := cfg.Claude.Subscription

	if subscription == "max" && cfg.Claude.MonthlyCostCents > 0 {
		// Fixed monthly subscription — prorate to daily
		totalCostCents = cfg.Claude.MonthlyCostCents / 30
		usageUnit = "tokens (Max $" + fmt.Sprintf("%.0f", float64(cfg.Claude.MonthlyCostCents)/100) + "/mo)"
	} else {
		// API pricing — estimate from token counts
		for model, usage := range modelUsage {
			pricing, ok := modelPricing[model]
			if !ok {
				pricing = modelPricing["claude-sonnet-4-6"]
			}
			inputCost := float64(usage.input) / 1_000_000 * pricing.input * 100
			outputCost := float64(usage.output) / 1_000_000 * pricing.output * 100
			totalCostCents += int(inputCost + outputCost)
		}
	}

	// Build raw data JSON
	rawData, _ := json.Marshal(map[string]interface{}{
		"input_tokens":                totalInputTokens,
		"output_tokens":               totalOutputTokens,
		"cache_creation_input_tokens":  totalCacheCreateTokens,
		"cache_read_input_tokens":      totalCacheReadTokens,
		"model_usage":                  modelUsage,
		"source":                       "claude_code_logs",
		"subscription":                 subscription,
	})

	entry := domain.CostEntry{
		Service:       "claude",
		PeriodStart:   since,
		PeriodEnd:     now,
		AmountCents:   totalCostCents,
		Currency:      "USD",
		UsageQuantity: float64(totalTokens),
		UsageUnit:     usageUnit,
		RawData:       string(rawData),
	}

	return s.SaveCostEntry(ctx, syncID, entry)
}

type parsedUsageEntry struct {
	model       string
	inputTokens int
	outputTokens int
	cacheCreate int
	cacheRead   int
}

func parseJSONLFile(path string, since time.Time) []parsedUsageEntry {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var results []parsedUsageEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large lines

	for scanner.Scan() {
		line := scanner.Bytes()

		// Quick check to avoid parsing lines without usage data
		if !strings.Contains(string(line), `"usage"`) {
			continue
		}

		var entry claudeLogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.Message.Usage == nil {
			continue
		}

		// Skip entries outside the time window
		if !entry.Timestamp.IsZero() && entry.Timestamp.Before(since) {
			continue
		}

		u := entry.Message.Usage
		results = append(results, parsedUsageEntry{
			model:       entry.Message.Model,
			inputTokens: u.InputTokens,
			outputTokens: u.OutputTokens,
			cacheCreate: u.CacheCreationInputTokens,
			cacheRead:   u.CacheReadInputTokens,
		})
	}

	return results
}

func init() {
	Register(&ClaudeCollector{})
}
