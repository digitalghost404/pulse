package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Pulse version",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag, _ := cmd.Flags().GetBool("json")
		if jsonFlag {
			enc := json.NewEncoder(os.Stdout)
			enc.Encode(map[string]string{"version": Version})
			return
		}
		fmt.Printf("pulse %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
