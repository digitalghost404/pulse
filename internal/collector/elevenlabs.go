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

type ElevenLabsCollector struct{}

func (e *ElevenLabsCollector) Name() string      { return "elevenlabs" }
func (e *ElevenLabsCollector) EnvVars() []string { return []string{"ELEVENLABS_API_KEY"} }

func (e *ElevenLabsCollector) Enabled(cfg *config.Config) bool {
	if !cfg.AdapterEnabled("elevenlabs") {
		return false
	}
	return os.Getenv("ELEVENLABS_API_KEY") != ""
}

type elevenLabsUsageResponse struct {
	Time  []int64              `json:"time"`
	Usage map[string][]float64 `json:"usage"`
}

func (e *ElevenLabsCollector) Collect(ctx context.Context, s store.Store, cfg *config.Config, syncID int64) error {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ELEVENLABS_API_KEY not set")
	}

	now := time.Now()
	startUnix := now.Add(-24 * time.Hour).UnixMilli()
	endUnix := now.UnixMilli()

	url := fmt.Sprintf("https://api.elevenlabs.io/v1/usage/character-stats?start_unix=%d&end_unix=%d&breakdown_type=none",
		startUnix, endUnix)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("xi-api-key", apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching ElevenLabs usage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ElevenLabs API returned %d (check ELEVENLABS_API_KEY)", resp.StatusCode)
	}

	body, err := limitedReadAll(resp.Body)
	if err != nil {
		return err
	}

	var usage elevenLabsUsageResponse
	if err := json.Unmarshal(body, &usage); err != nil {
		return fmt.Errorf("parsing ElevenLabs usage: %w", err)
	}

	// Sum all character usage across all breakdowns
	var totalChars float64
	for _, counts := range usage.Usage {
		for _, c := range counts {
			totalChars += c
		}
	}

	amountCents := 0
	if cfg.Costs.Pricing.ElevenLabsCentsPer1KChars > 0 {
		amountCents = int(totalChars / 1000 * float64(cfg.Costs.Pricing.ElevenLabsCentsPer1KChars))
	}

	// Build raw data as proper JSON combining usage and subscription info
	rawData := e.buildRawData(ctx, apiKey, body)

	entry := domain.CostEntry{
		Service:       "elevenlabs",
		PeriodStart:   now.Add(-24 * time.Hour),
		PeriodEnd:     now,
		AmountCents:   amountCents,
		Currency:      "USD",
		UsageQuantity: totalChars,
		UsageUnit:     "characters",
		RawData:       rawData,
	}

	return s.SaveCostEntry(ctx, syncID, entry)
}

type elevenLabsSubscription struct {
	Tier                 string `json:"tier"`
	CharacterCount       int64  `json:"character_count"`
	CharacterLimit       int64  `json:"character_limit"`
	NextCharacterCountAt string `json:"next_character_count_reset_unix"`
}

type elevenLabsRawData struct {
	Usage        json.RawMessage         `json:"usage"`
	Subscription *elevenLabsSubscription `json:"subscription,omitempty"`
}

func (e *ElevenLabsCollector) buildRawData(ctx context.Context, apiKey string, usageBody []byte) string {
	raw := elevenLabsRawData{Usage: usageBody}

	sub, err := e.fetchSubscription(ctx, apiKey)
	if err == nil {
		raw.Subscription = sub
	}

	data, _ := json.Marshal(raw)
	return string(data)
}

func (e *ElevenLabsCollector) fetchSubscription(ctx context.Context, apiKey string) (*elevenLabsSubscription, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.elevenlabs.io/v1/user/subscription", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("xi-api-key", apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ElevenLabs subscription API returned %d", resp.StatusCode)
	}

	body, err := limitedReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var sub elevenLabsSubscription
	if err := json.Unmarshal(body, &sub); err != nil {
		return nil, err
	}

	return &sub, nil
}

func init() {
	Register(&ElevenLabsCollector{})
}
