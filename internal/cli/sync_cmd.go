package cli

import (
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
	collectors := collector.Enabled(cfg)

	only, _ := cmd.Flags().GetString("only")

	var result psync.Result
	if only != "" {
		result = engine.RunOnly(cmd.Context(), collectors, only)
	} else {
		result = engine.Run(cmd.Context(), collectors)
	}

	for _, e := range result.Errors {
		log.Printf("WARN: %s", e)
	}

	fmt.Fprintf(os.Stderr, "sync: %s (run %d)\n", result.Status, result.SyncID)

	switch result.Status {
	case "success":
		return nil
	case "partial":
		os.Exit(1)
	default:
		os.Exit(2)
	}
	return nil
}
