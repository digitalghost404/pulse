package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Pulse configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate default config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.DefaultConfigPath()
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config already exists at %s", path)
		}
		if err := config.GenerateDefault(path); err != nil {
			return err
		}
		fmt.Printf("Config created at %s\n", path)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print current config",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.DefaultConfigPath()
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading config: %w (run 'pulse config init' to create)", err)
		}
		fmt.Print(string(data))
		return nil
	},
}

var configAdaptersCmd = &cobra.Command{
	Use:   "adapters",
	Short: "Show adapter status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		fmt.Printf("%-15s %-10s %-10s %s\n", "ADAPTER", "ENABLED", "ENV OK", "ENV VARS")
		fmt.Println(strings.Repeat("-", 60))

		for _, c := range collector.All() {
			enabled := cfg.AdapterEnabled(c.Name())
			enabledStr := "yes"
			if !enabled {
				enabledStr = "no"
			}

			envVars := c.EnvVars()
			envOK := "n/a"
			envList := "-"
			if len(envVars) > 0 {
				envList = strings.Join(envVars, ", ")
				allSet := true
				for _, v := range envVars {
					if os.Getenv(v) == "" {
						allSet = false
						break
					}
				}
				if allSet {
					envOK = "yes"
				} else {
					envOK = "MISSING"
				}
			}

			fmt.Printf("%-15s %-10s %-10s %s\n", c.Name(), enabledStr, envOK, envList)
		}
		return nil
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configAdaptersCmd)
	rootCmd.AddCommand(configCmd)
}
