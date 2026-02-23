package cli

import (
	"fmt"
	"os"

	"github.com/qonto/pgctl/pkg/pgctl"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run commands, usually prepared by an associated init command.",
	}
	runCmd.AddCommand(runRelocationCmd())

	return runCmd
}

func runRelocationCmd() *cobra.Command {
	var source, target string
	var apply, DDLAreFrozen, WritesAreFrozen bool

	runRelocationCmd := &cobra.Command{
		Use:   "relocation",
		Short: "Run a relocation process of database after initialized by init cmd in a single schema",
		Long: `Run a relocation process of database by running all necessary checks in order, then running the following steps:
			1. wait for lag zero between source publication and target subscription
			2. copy sequences from source to target
			3. drop publication on source alias (from)
			4. drop subscription on target alias (to)

			Disclaimer: this command doesn't support yet databases using multiple schemas
		`,
		PreRun: func(cmd *cobra.Command, args []string) {
			if !DDLAreFrozen {
				fmt.Printf("❌ DDL should be frozen for all clients to run relocation run\nOnce done, confirm with --no-ddl-confirmed flag")
				os.Exit(1)
			}
			if !WritesAreFrozen {
				fmt.Printf("❌ Writes should be disabled on all clients to run relocation run\nOnce done, confirm with --no-writes-confirmed flag")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			err = app.RunRelocation(source, target, apply)
			if err != nil {
				fmt.Printf("%v", err)
				os.Exit(1)
			}
		},
	}

	runRelocationCmd.Flags().StringVar(&source, "from", "", "source database alias")
	runRelocationCmd.Flags().StringVar(&target, "to", "", "target database alias")
	runRelocationCmd.Flags().BoolVar(&apply, "apply", false, "if set, will apply the changes")
	runRelocationCmd.Flags().BoolVarP(&DDLAreFrozen, "no-ddl-confirmed", "", false, "confirm that DDL are frozen for the application writing to this database")
	runRelocationCmd.Flags().BoolVarP(&WritesAreFrozen, "no-writes-confirmed", "", false, "confirm that writes are frozen for the application using this database")
	runRelocationCmd.MarkPersistentFlagRequired("from") //nolint: errcheck,gosec
	runRelocationCmd.MarkPersistentFlagRequired("to")   //nolint: errcheck,gosec

	return runRelocationCmd
}
