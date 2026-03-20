package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/collector"
	psync "github.com/xcoleman/pulse/internal/sync"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Collect data from all sources",
	RunE:  runSync,
}

func init() {
	syncCmd.Flags().String("only", "", "Run only a specific collector")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	s, err := openStore()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer s.Close()

	engine := psync.NewEngine(s, cfg)
	allCollectors := collector.All()
	enabledCollectors := collector.Enabled(cfg)

	only, _ := cmd.Flags().GetString("only")

	var result psync.Result
	if only != "" {
		// Check if the collector exists but is disabled
		found := false
		for _, c := range allCollectors {
			if c.Name() == only {
				found = true
				if !c.Enabled(cfg) {
					envVars := c.EnvVars()
					if len(envVars) > 0 {
						return fmt.Errorf("collector %q is disabled — missing env var(s): %s", only, fmt.Sprint(envVars))
					}
					return fmt.Errorf("collector %q is disabled in config", only)
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("collector %q not found — available: %s", only, collectorNames(allCollectors))
		}
		result = engine.RunOnly(cmd.Context(), enabledCollectors, only)
	} else {
		result = engine.Run(cmd.Context(), enabledCollectors)
	}

	if !verbose {
		for _, e := range result.Errors {
			log.Printf("WARN: %s", e)
		}
	}

	jsonFlag, _ := cmd.Flags().GetBool("json")
	if jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Fprintf(os.Stderr, "sync: %s (run %d)\n", result.Status, result.SyncID)

	switch result.Status {
	case "partial":
		return fmt.Errorf("sync partial: %d collector(s) failed", len(result.Errors))
	case "failed":
		return fmt.Errorf("sync failed: all collectors failed")
	}
	return nil
}

func collectorNames(collectors []collector.Collector) string {
	names := make([]string, len(collectors))
	for i, c := range collectors {
		names[i] = c.Name()
	}
	return fmt.Sprint(names)
}
