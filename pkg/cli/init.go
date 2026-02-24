package cli

import (
	"fmt"
	"os"

	"github.com/qonto/pgctl/pkg/pgctl"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Various initialization tools that prepare operation before a run command.",
	}
	initCmd.AddCommand(initRelocationCmd())

	return initCmd
}

func initRelocationCmd() *cobra.Command {
	var source, target string
	var apply, DDLAreFrozen bool

	initRelocationCmd := &cobra.Command{
		Use:   "relocation",
		Short: "Initialize a relocation process of database",
		Long: `Initialize a relocation process of database by running all necessary checks in order, then running the following steps:
			1. copy schema from source to target alias
			2. create publication with all table from source schema
			3. create subscription to created publication on target alias`,
		PreRun: func(cmd *cobra.Command, args []string) {
			if !DDLAreFrozen {
				fmt.Printf("❌ DDL should be frozen for all clients to run relocation initialization\nOnce done, confirm with --no-ddl-confirmed flag")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			err = app.InitRelocation(source, target, apply)
			if err != nil {
				fmt.Printf("%v", err)
				os.Exit(1)
			}
		},
	}

	initRelocationCmd.Flags().StringVar(&source, "from", "", "source database alias")
	initRelocationCmd.Flags().StringVar(&target, "to", "", "target database alias")
	initRelocationCmd.Flags().BoolVar(&apply, "apply", false, "if set, will apply the changes")
	initRelocationCmd.Flags().BoolVarP(&DDLAreFrozen, "no-ddl-confirmed", "", false, "confirm that DDL are frozen for the application writing to this database")
	initRelocationCmd.MarkPersistentFlagRequired("from") //nolint: errcheck,gosec
	initRelocationCmd.MarkPersistentFlagRequired("to")   //nolint: errcheck,gosec

	return initRelocationCmd
}
