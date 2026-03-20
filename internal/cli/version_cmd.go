package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Pulse version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("pulse %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
