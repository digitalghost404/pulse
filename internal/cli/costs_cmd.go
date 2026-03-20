package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/briefing"
)

var costsCmd = &cobra.Command{
	Use:   "costs",
	Short: "Print cost summary",
	RunE:  runCosts,
}

func init() {
	costsCmd.Flags().String("service", "", "Filter to a specific service")
	costsCmd.Flags().String("period", "", "Time period (e.g., 7d, 30d)")
	rootCmd.AddCommand(costsCmd)
}

func runCosts(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	period, _ := cmd.Flags().GetString("period")
	if period != "" {
		cfg.Costs.DefaultPeriod = period
	}

	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	engine := briefing.NewEngine(s, cfg)
	b, err := engine.Build(cmd.Context())
	if err != nil {
		return err
	}

	service, _ := cmd.Flags().GetString("service")
	cs := b.CostSummary

	jsonFlag, _ := cmd.Flags().GetBool("json")

	if service != "" {
		// Filter
		for _, sc := range cs.ByService {
			if sc.Service == service {
				if jsonFlag {
					return jsonOut(sc)
				}
				fmt.Printf("%s: $%.2f (%.0f %s)\n", sc.Service, float64(sc.AmountCents)/100, sc.UsageQuantity, sc.UsageUnit)
				return nil
			}
		}
		return fmt.Errorf("service %q not found", service)
	}

	if jsonFlag {
		return jsonOut(cs)
	}

	fmt.Printf("Costs (%s)\n\n", cs.Period)
	for _, sc := range cs.ByService {
		fmt.Printf("  %s: $%.2f\n", sc.Service, float64(sc.AmountCents)/100)
	}
	fmt.Printf("\n  Total: $%.2f — Burn: $%.2f/day\n",
		float64(cs.TotalCents)/100, float64(cs.BurnRateCents)/100)

	return nil
}
