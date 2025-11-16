package cmd

import (
	"fmt"
	"os"

	"github.com/lukasbarti/musixx/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "musixx",
	Short: "musixx manages and plays your local music library",
	Long:  "musixx runs a local-first music library with a web UI for playback and management.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Load(); err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("musixx CLI coming soon — web server placeholder listening on port %d.\n", config.HTTPPort())
	},
}

func init() {
	rootCmd.AddCommand(newServeCmd())
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
