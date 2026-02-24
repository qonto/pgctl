package cli

import (
	"fmt"
	"os"

	"github.com/qonto/pgctl/pkg/pgctl"
	"github.com/spf13/cobra"
)

func copyCmd() *cobra.Command {
	copyCmd := &cobra.Command{
		Use:   "copy",
		Short: "Various copy tools.",
	}
	copyCmd.AddCommand(copySchemaCmd())
	copyCmd.AddCommand(copySequencesCmd())

	copyCmd.PersistentFlags().BoolVar(&runApply, "apply", false, "if set, removes DRY RUN MODE and actually runs command")

	return copyCmd
}

func copySchemaCmd() *cobra.Command {
	var source, target string
	var allTables bool

	copySchemaCmd := &cobra.Command{
		Use:   "schema",
		Short: "Copy a schema from one database to another",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.CopySchema(source, target, allTables, runApply)
		},
	}

	copySchemaCmd.Flags().StringVar(&source, "from", "", "source database alias")
	copySchemaCmd.Flags().StringVar(&target, "to", "", "target database alias")
	copySchemaCmd.Flags().BoolVar(&allTables, "all-tables", false, "copy all tables")

	copySchemaCmd.MarkFlagRequired("from") //nolint: errcheck,gosec
	copySchemaCmd.MarkFlagRequired("to")   //nolint: errcheck,gosec

	return copySchemaCmd
}

func copySequencesCmd() *cobra.Command {
	var source, target string

	copySequencesCmd := &cobra.Command{
		Use:   "sequences",
		Short: "Copy sequences from the source and import them to the target database. It'll overwrite/update existing sequences.",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.CopySequences(source, target, runApply)
		},
	}
	copySequencesCmd.Flags().StringVar(&source, "from", "", "source database alias")
	copySequencesCmd.Flags().StringVar(&target, "to", "", "target database alias")
	copySequencesCmd.MarkFlagRequired("from") //nolint: errcheck,gosec
	copySequencesCmd.MarkFlagRequired("to")   //nolint: errcheck,gosec
	return copySequencesCmd
}
