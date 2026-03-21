package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/briefing"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
	"github.com/xcoleman/pulse/internal/writer"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "pulse",
	Short: "Personal command center — briefing, project health, cost tracking",
	Long:  "Pulse synthesizes signals from your projects, AI services, and dev environment into a single morning briefing.",
	RunE:  runBriefing,
}

// exitCode allows commands to set a specific exit code (e.g., 1 for partial sync, 2 for total failure).
var exitCode int

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug output")
	rootCmd.PersistentFlags().Bool("json", false, "Output as JSON")
	rootCmd.Flags().String("since", "", "Show data since duration (e.g., 24h, 7d)")

	// Set up verbose logging
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if !verbose {
			log.SetOutput(io.Discard)
		} else {
			log.SetOutput(os.Stderr)
			log.SetFlags(log.Ltime)
		}
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if exitCode == 0 {
			exitCode = 1
		}
		os.Exit(exitCode)
	}
}

func loadConfig() (*config.Config, error) {
	return config.Load(config.DefaultConfigPath())
}

func openStore() (store.Store, error) {
	dbPath := config.DefaultConfigDir() + "/pulse.db"
	return store.NewSQLite(dbPath)
}

func runBriefing(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	s, err := openStore()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer s.Close()

	engine := briefing.NewEngine(s, cfg)

	sinceStr, _ := cmd.Flags().GetString("since")
	var opts briefing.BuildOptions
	if sinceStr != "" {
		d, err := parseSinceDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
		opts.Since = time.Now().Add(-d)
	}

	b, err := engine.BuildWithOptions(cmd.Context(), opts)
	if err != nil {
		return err
	}

	jsonFlag, _ := cmd.Flags().GetBool("json")

	// Render to buffer for both stdout and history
	var rendered bytes.Buffer
	w := writer.NewStdoutWriter(&rendered)
	w.Write(cmd.Context(), b, cfg)

	if jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		// Save history even for JSON output
		s.SaveBriefing(cmd.Context(), domain.BriefingEntry{
			CreatedAt: time.Now(),
			Content:   rendered.String(),
			Writer:    "stdout-json",
		})
		return enc.Encode(b)
	}

	// Write to stdout
	fmt.Print(rendered.String())

	// Save history
	s.SaveBriefing(cmd.Context(), domain.BriefingEntry{
		CreatedAt: time.Now(),
		Content:   rendered.String(),
		Writer:    "stdout",
	})

	return nil
}

func parseSinceDuration(s string) (time.Duration, error) {
	// Support "Nd" format (e.g., "7d" → 168h)
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(numStr, "%d", &days); err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
