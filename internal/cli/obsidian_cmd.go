package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/briefing"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/writer"
)

var obsidianCmd = &cobra.Command{
	Use:   "obsidian",
	Short: "Append briefing to today's Obsidian daily note",
	RunE:  runObsidian,
}

func init() {
	obsidianCmd.Flags().String("since", "", "Show data since duration (e.g., 24h, 7d)")
	rootCmd.AddCommand(obsidianCmd)
}

func runObsidian(cmd *cobra.Command, args []string) error {
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

	jsonFlag, _ := cmd.Flags().GetBool("json")
	if jsonFlag {
		// Save history even for JSON output
		s.SaveBriefing(cmd.Context(), domain.BriefingEntry{
			CreatedAt: time.Now(),
			Content:   b.GeneratedAt.Format(time.RFC3339),
			Writer:    "obsidian-json",
		})
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(b)
	}

	w := writer.NewObsidianWriter()
	if err := w.Write(cmd.Context(), b, cfg); err != nil {
		return err
	}

	// Render content for history
	var rendered bytes.Buffer
	sw := writer.NewStdoutWriter(&rendered)
	sw.Write(cmd.Context(), b, cfg)

	// Save history
	s.SaveBriefing(cmd.Context(), domain.BriefingEntry{
		CreatedAt: time.Now(),
		Content:   rendered.String(),
		Writer:    "obsidian",
	})

	notePath := cfg.ObsidianDailyNotePath(b.GeneratedAt)
	fmt.Printf("Briefing written to %s\n", notePath)
	return nil
}
