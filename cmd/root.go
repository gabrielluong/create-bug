package cmd

import (
	"fmt"
	"os"

	"github.com/gabrielluong/create-bug/internal/update"
	"github.com/spf13/cobra"
)

const version = "0.5.1"

var updateNotice string

var rootCmd = &cobra.Command{
	Use:     "create-bug [summary]",
	Short:   "Create a Bugzilla bug",
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if latest, outdated := update.CheckCached(version); outdated {
			updateNotice = fmt.Sprintf("A new version (%s) is available. Run: go install github.com/gabrielluong/create-bug@latest", latest)
		}
		go update.RefreshCache()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if updateNotice != "" {
			fmt.Fprintln(os.Stderr, updateNotice)
		}
	},
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
