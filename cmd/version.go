package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of mdpress",
	Long:  "Display the version, build time, and other build information for mdpress.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("mdpress version %s\n", Version)
		if BuildTime != "unknown" && BuildTime != "" {
			fmt.Printf("Built at %s\n", BuildTime)
		}
	},
}
