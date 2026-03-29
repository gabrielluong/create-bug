package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "create-bug [summary]",
	Short:   "Create a Bugzilla bug",
	Version: "0.1.0",
}

func init() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	setupCreateBugCmd(rootCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
