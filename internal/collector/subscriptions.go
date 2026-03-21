package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

type SubscriptionsCollector struct{}

func (s *SubscriptionsCollector) Name() string      { return "subscriptions" }
func (s *SubscriptionsCollector) EnvVars() []string { return nil }

func (s *SubscriptionsCollector) Enabled(cfg *config.Config) bool {
	return len(cfg.Costs.Subscriptions) > 0
}

func (s *SubscriptionsCollector) Collect(ctx context.Context, st store.Store, cfg *config.Config, syncID int64) error {
	now := time.Now()

	for _, sub := range cfg.Costs.Subscriptions {
		dailyCents := sub.MonthlyCostCents / 30

		rawData, _ := json.Marshal(map[string]interface{}{
			"name":         sub.Name,
			"monthly_cost": sub.MonthlyCostCents,
			"notes":        sub.Notes,
			"source":       "config_subscription",
		})

		entry := domain.CostEntry{
			Service:       sub.Service,
			PeriodStart:   now.Add(-24 * time.Hour),
			PeriodEnd:     now,
			AmountCents:   dailyCents,
			Currency:      "USD",
			UsageQuantity: 1,
			UsageUnit:     sub.Name + " ($" + formatDollars(sub.MonthlyCostCents) + "/mo)",
			RawData:       string(rawData),
		}

		if err := st.SaveCostEntry(ctx, syncID, entry); err != nil {
			return err
		}
	}

	return nil
}

func formatDollars(cents int) string {
	if cents%100 == 0 {
		return fmt.Sprintf("%d", cents/100)
	}
	return fmt.Sprintf("%.2f", float64(cents)/100)
}

func init() {
	Register(&SubscriptionsCollector{})
}
