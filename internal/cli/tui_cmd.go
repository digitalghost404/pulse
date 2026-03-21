package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/briefing"
	"github.com/xcoleman/pulse/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive dashboard",
	RunE:  runTUI,
}

func init() {
	tuiCmd.Flags().String("since", "", "Show data since duration (e.g., 24h, 7d)")
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	s, err := openStore()
	if err != nil {
		return err
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

	// --json skips the TUI and prints briefing data
	jsonFlag, _ := cmd.Flags().GetBool("json")
	if jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(b)
	}

	model := tui.NewModel(b)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
