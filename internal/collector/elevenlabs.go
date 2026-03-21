package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching ElevenLabs usage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ElevenLabs API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
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

	entry := domain.CostEntry{
		Service:       "elevenlabs",
		PeriodStart:   now.Add(-24 * time.Hour),
		PeriodEnd:     now,
		AmountCents:   amountCents,
		Currency:      "USD",
		UsageQuantity: totalChars,
		UsageUnit:     "characters",
		RawData:       string(body),
	}

	// Also try to get the subscription info for quota context
	subEntry, err := e.fetchSubscription(ctx, apiKey, now)
	if err == nil && subEntry != nil {
		entry.RawData = string(body) + "\n" + subEntry.RawData
	}

	return s.SaveCostEntry(ctx, syncID, entry)
}

type elevenLabsSubscription struct {
	Tier                 string `json:"tier"`
	CharacterCount       int64  `json:"character_count"`
	CharacterLimit       int64  `json:"character_limit"`
	NextCharacterCountAt string `json:"next_character_count_reset_unix"`
}

func (e *ElevenLabsCollector) fetchSubscription(ctx context.Context, apiKey string, now time.Time) (*domain.CostEntry, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.elevenlabs.io/v1/user/subscription", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("xi-api-key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ElevenLabs subscription API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var sub elevenLabsSubscription
	if err := json.Unmarshal(body, &sub); err != nil {
		return nil, err
	}

	return &domain.CostEntry{
		RawData: `{"subscription":` + string(body) + `,"used":"` + strconv.FormatInt(sub.CharacterCount, 10) + `","limit":"` + strconv.FormatInt(sub.CharacterLimit, 10) + `"}`,
	}, nil
}

func init() {
	Register(&ElevenLabsCollector{})
}
