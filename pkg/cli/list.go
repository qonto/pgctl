package cli

import (
	"fmt"
	"os"

	"github.com/qonto/pgctl/pkg/pgctl"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Various listing tools.",
	}
	listCmd.AddCommand(listDatabasesCmd())
	listCmd.AddCommand(listExtensionsCmd())
	listCmd.AddCommand(listTablesCmd())
	listCmd.AddCommand(listPublicationsCmd())
	listCmd.AddCommand(listSequencesCmd())
	listCmd.AddCommand(listSubscriptionsCmd())

	listCmd.PersistentFlags().StringVar(&database, "on", "", "selected alias")
	listCmd.MarkPersistentFlagRequired("on") //nolint: errcheck,gosec

	return listCmd
}

func listDatabasesCmd() *cobra.Command {
	listDatabasesCmd := &cobra.Command{
		Use:   "databases",
		Short: "List databases from this alias.",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}
			app.ListDatabases(database)
		},
	}
	return listDatabasesCmd
}

func listExtensionsCmd() *cobra.Command {
	listExtensionsCmd := &cobra.Command{
		Use:   "extensions",
		Short: "List installed extensions on the server.",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.ListExtensions(database, runOnAllDatabases)
		},
	}
	listExtensionsCmd.PersistentFlags().BoolVar(&runOnAllDatabases, "all-databases", false, "if set, will run on all databases on the server using the alias provided")
	return listExtensionsCmd
}

func listTablesCmd() *cobra.Command {
	var addSchemaPrefix bool

	listTablesCmd := &cobra.Command{
		Use:   "tables",
		Short: "List tables in the database.",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			_ = app.ListTables(database, addSchemaPrefix)
		},
	}
	listTablesCmd.PersistentFlags().BoolVar(&addSchemaPrefix, "with-schema-prefix", false, "if set, will prefix schema to result (ex: public.tablename)")

	return listTablesCmd
}

func listPublicationsCmd() *cobra.Command {
	var runOnAllDatabases bool

	listPublicationsCmd := &cobra.Command{
		Use:   "publications",
		Short: "List publications on the server.",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.ListPublications(database, runOnAllDatabases)
		},
	}

	listPublicationsCmd.PersistentFlags().BoolVar(&runOnAllDatabases, "all-databases", false, "if set, will run on all databases on the server using the alias provided")

	return listPublicationsCmd
}

func listSequencesCmd() *cobra.Command {
	listSequencesCmd := &cobra.Command{
		Use:   "sequences",
		Short: "List sequences in a database.",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}
			app.ListSequences(database)
		},
	}
	return listSequencesCmd
}

func listSubscriptionsCmd() *cobra.Command {
	var runOnAllDatabases bool

	listSubscriptionsCmd := &cobra.Command{
		Use:   "subscriptions",
		Short: "List subscriptions on the server.",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.ListSubscriptions(database, runOnAllDatabases)
		},
	}

	listSubscriptionsCmd.PersistentFlags().BoolVar(&runOnAllDatabases, "all-databases", false, "if set, will run on all databases on the server using the alias provided")

	return listSubscriptionsCmd
}
