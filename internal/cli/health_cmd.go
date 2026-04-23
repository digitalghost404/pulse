package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check Pulse health — config and database connectivity",
	RunE:  runHealth,
}

func init() {
	rootCmd.AddCommand(healthCmd)
}

func runHealth(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := cmd.Flags().GetBool("json")

	type checkResult struct {
		Name    string `json:"name"`
		Status  string `json:"status"`
		Message string `json:"message,omitempty"`
	}

	var checks []checkResult
	healthy := true

	// Check config
	cfg, err := loadConfig()
	if err != nil {
		checks = append(checks, checkResult{
			Name:    "config",
			Status:  "FAIL",
			Message: err.Error(),
		})
		healthy = false
	} else {
		scanDirs := cfg.Projects.Scan
		if len(scanDirs) == 0 {
			checks = append(checks, checkResult{
				Name:    "config",
				Status:  "WARN",
				Message: "no project scan directories configured",
			})
		} else {
			checks = append(checks, checkResult{
				Name:    "config",
				Status:  "OK",
				Message: fmt.Sprintf("%d scan dir(s) configured", len(scanDirs)),
			})
		}
	}

	// Check database
	s, err := openStore()
	if err != nil {
		checks = append(checks, checkResult{
			Name:    "database",
			Status:  "FAIL",
			Message: err.Error(),
		})
		healthy = false
	} else {
		defer s.Close()

		syncID, err := s.LatestSyncID(cmd.Context())
		if err != nil {
			checks = append(checks, checkResult{
				Name:    "database",
				Status:  "FAIL",
				Message: err.Error(),
			})
			healthy = false
		} else if syncID == 0 {
			checks = append(checks, checkResult{
				Name:    "database",
				Status:  "WARN",
				Message: "connected, no sync runs yet",
			})
		} else {
			checks = append(checks, checkResult{
				Name:    "database",
				Status:  "OK",
				Message: fmt.Sprintf("connected, latest sync_id=%d", syncID),
			})
		}
	}

	// Set exit code
	if !healthy {
		exitCode = 2
	} else {
		for _, c := range checks {
			if c.Status == "WARN" {
				exitCode = 1
				break
			}
		}
	}

	if jsonFlag {
		type healthResponse struct {
			Status string        `json:"status"`
			Checks []checkResult `json:"checks"`
		}
		status := "healthy"
		if exitCode == 1 {
			status = "degraded"
		} else if exitCode == 2 {
			status = "unhealthy"
		}
		return jsonOut(healthResponse{
			Status: status,
			Checks: checks,
		})
	}

	// Human-readable output
	for _, c := range checks {
		icon := "✓"
		if c.Status == "WARN" {
			icon = "!"
		} else if c.Status == "FAIL" {
			icon = "✗"
		}
		msg := c.Message
		if msg == "" {
			msg = c.Status
		}
		fmt.Printf("[%s] %s: %s\n", icon, c.Name, msg)
	}

	statusLine := "Status: healthy"
	if exitCode == 1 {
		statusLine = "Status: degraded"
	} else if exitCode == 2 {
		statusLine = "Status: unhealthy"
	}
	fmt.Println(statusLine)

	return nil
}
