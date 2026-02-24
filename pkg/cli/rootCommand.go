package cli

import (
	"github.com/spf13/cobra"
)

var (
	database          string //nolint: gochecknoglobals
	runApply          bool   //nolint: gochecknoglobals
	runOnAllDatabases bool   //nolint: gochecknoglobals
)

func RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "pgctl",
		Short: "A set of commands to manage PostgreSQL maintenance operations.",
	}

	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(pingCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(runCmd())
	rootCmd.AddCommand(copyCmd())
	rootCmd.AddCommand(createCmd())
	rootCmd.AddCommand(dropCmd())
	rootCmd.AddCommand(checkCmd())
	rootCmd.AddCommand(versionCmd())

	return rootCmd
}
