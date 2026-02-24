package cli

import (
	"fmt"
	"os"

	"github.com/qonto/pgctl/pkg/pgctl"
	"github.com/spf13/cobra"
)

func updateCmd() *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Various update tools.",
	}
	updateCmd.AddCommand(updateExtensionsCmd())

	updateCmd.PersistentFlags().StringVar(&database, "on", "", "selected alias")
	updateCmd.PersistentFlags().BoolVar(&runApply, "apply", false, "if set, removes DRY RUN MODE and actually runs command")
	updateCmd.MarkPersistentFlagRequired("on") //nolint: errcheck,gosec

	return updateCmd
}

func updateExtensionsCmd() *cobra.Command {
	var runMajorUpdates bool

	updateExtensionsCmd := &cobra.Command{
		Use:   "extensions",
		Short: "Update installed extensions on the server.",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.UpdateExtensions(database, runOnAllDatabases, runMajorUpdates, runApply)
		},
	}
	updateExtensionsCmd.PersistentFlags().BoolVar(&runOnAllDatabases, "all-databases", false, "if set, will run on all databases on the server using the alias provided")
	updateExtensionsCmd.PersistentFlags().BoolVar(&runMajorUpdates, "include-major-versions", false, "if set, will also run major updates")
	return updateExtensionsCmd
}
