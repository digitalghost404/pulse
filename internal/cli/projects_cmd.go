package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/briefing"
	"github.com/xcoleman/pulse/internal/domain"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Print project health summary",
	RunE:  runProjects,
}

func init() {
	projectsCmd.Flags().String("repo", "", "Filter to a specific repo")
	rootCmd.AddCommand(projectsCmd)
}

func runProjects(cmd *cobra.Command, args []string) error {
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

	repo, _ := cmd.Flags().GetString("repo")
	jsonFlag, _ := cmd.Flags().GetBool("json")

	if repo != "" {
		for _, p := range b.Projects {
			if p.RepoName == repo {
				if jsonFlag {
					return jsonOut(p)
				}
				printProject(p)
				return nil
			}
		}
		return fmt.Errorf("repo %q not found", repo)
	}

	if jsonFlag {
		return jsonOut(b.Projects)
	}

	for _, p := range b.Projects {
		printProject(p)
	}
	return nil
}

func printProject(p domain.ProjectSummary) {
	icon := "✓"
	details := "clean"
	if p.DirtyFiles > 0 || p.Ahead > 0 || p.Behind > 0 {
		icon = "⚠"
		var parts []string
		if p.DirtyFiles > 0 {
			parts = append(parts, fmt.Sprintf("%d dirty", p.DirtyFiles))
		}
		if p.Ahead > 0 {
			parts = append(parts, fmt.Sprintf("%d ahead", p.Ahead))
		}
		if p.Behind > 0 {
			parts = append(parts, fmt.Sprintf("%d behind", p.Behind))
		}
		details = strings.Join(parts, ", ")
	}
	fmt.Printf("  %s %s (%s) — %s\n", icon, p.RepoName, p.Branch, details)

	if len(p.Branches) > 1 {
		for _, br := range p.Branches {
			if !br.IsCurrent {
				merged := ""
				if br.IsMerged {
					merged = " [merged]"
				}
				fmt.Printf("      ↳ %s%s\n", br.BranchName, merged)
			}
		}
	}
}

func jsonOut(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
