package cli

import (
	"fmt"
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
	b, err := engine.Build(cmd.Context())
	if err != nil {
		return err
	}

	w := writer.NewObsidianWriter()
	if err := w.Write(cmd.Context(), b, cfg); err != nil {
		return err
	}

	// Save history
	s.SaveBriefing(cmd.Context(), domain.BriefingEntry{
		CreatedAt: time.Now(),
		Content:   b.GeneratedAt.String(),
		Writer:    "obsidian",
	})

	notePath := cfg.ObsidianDailyNotePath(b.GeneratedAt)
	fmt.Printf("Briefing written to %s\n", notePath)
	return nil
}
