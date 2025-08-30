package cmd

import (
	"fmt"

	"github.com/containifyci/dunebot/pkg/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Print the version, commit, and build date information for dunebot.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\n", version.GetVersion())
		fmt.Printf("Commit:  %s\n", version.GetCommit())
		fmt.Printf("Date:    %s\n", version.GetDate())
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}