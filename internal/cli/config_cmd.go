// Package cli provides Cobra commands for the Pulse CLI.
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

		jsonFlag, _ := cmd.Flags().GetBool("json")
		if jsonFlag {
			return jsonOut(map[string]string{"path": path, "status": "created"})
		}
		fmt.Printf("Config created at %s\n", path)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print current config",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonFlag, _ := cmd.Flags().GetBool("json")

		if jsonFlag {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			return jsonOut(cfg)
		}

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

		jsonFlag, _ := cmd.Flags().GetBool("json")

		type adapterInfo struct {
			Name    string   `json:"name"`
			Enabled bool     `json:"enabled"`
			EnvOK   string   `json:"env_ok"`
			EnvVars []string `json:"env_vars,omitempty"`
		}

		var adapters []adapterInfo

		for _, c := range collector.All() {
			enabled := cfg.AdapterEnabled(c.Name())
			envVars := c.EnvVars()
			envOK := "n/a"
			if len(envVars) > 0 {
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
			adapters = append(adapters, adapterInfo{
				Name:    c.Name(),
				Enabled: enabled,
				EnvOK:   envOK,
				EnvVars: envVars,
			})
		}

		if jsonFlag {
			return jsonOut(adapters)
		}

		fmt.Printf("%-15s %-10s %-10s %s\n", "ADAPTER", "ENABLED", "ENV OK", "ENV VARS")
		fmt.Println(strings.Repeat("-", 60))

		for _, a := range adapters {
			enabledStr := "yes"
			if !a.Enabled {
				enabledStr = "no"
			}
			envList := "-"
			if len(a.EnvVars) > 0 {
				envList = strings.Join(a.EnvVars, ", ")
			}
			fmt.Printf("%-15s %-10s %-10s %s\n", a.Name, enabledStr, a.EnvOK, envList)
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
