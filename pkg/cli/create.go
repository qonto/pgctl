package cli

import (
	"fmt"
	"os"

	"github.com/qonto/pgctl/pkg/pgctl"
	"github.com/spf13/cobra"
)

func createCmd() *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Various creation tools.",
	}
	createCmd.AddCommand(createPublicationCmd())
	createCmd.AddCommand(createSubscriptionCmd())

	return createCmd
}

func createPublicationCmd() *cobra.Command {
	var publicationTables []string
	var runOnAllTables bool
	var apply bool

	createPublicationCmd := &cobra.Command{
		Use:   "publication",
		Short: "Create a publication",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			if runOnAllTables {
				tables, err := app.GetTables(database)
				if err != nil {
					fmt.Printf("Unable to get tables: %v\n", err)
					os.Exit(1)
				}

				_ = app.CreatePublication(database, tables, apply)
				return
			}

			if len(publicationTables) > 0 {
				_ = app.CreatePublication(database, publicationTables, apply)
				return
			}

			selectedTables, err := app.SelectTables(database, "Select tables to create publication on")
			if err != nil {
				fmt.Printf("Unable to select tables: %v\n", err)
				os.Exit(1)
			}

			_ = app.CreatePublication(database, selectedTables, apply)
		},
	}

	createPublicationCmd.PersistentFlags().StringVar(&database, "on", "", "selected alias")
	createPublicationCmd.MarkPersistentFlagRequired("on") //nolint: errcheck,gosec

	createPublicationCmd.Flags().StringSliceVar(&publicationTables, "tables", []string{}, "tables to include in the publication")
	createPublicationCmd.Flags().BoolVar(&runOnAllTables, "all-tables", false, "if set, will run on all tables on the alias provided")
	createPublicationCmd.Flags().BoolVar(&apply, "apply", false, "if set, will apply the changes")

	return createPublicationCmd
}

func createSubscriptionCmd() *cobra.Command {
	var fromDatabase string
	var publicationName string
	var apply bool

	createSubscriptionCmd := &cobra.Command{
		Use:   "subscription",
		Short: "Create a subscription from a publication",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.CreateSubscription(database, fromDatabase, publicationName, apply)
		},
	}

	createSubscriptionCmd.PersistentFlags().StringVar(&database, "on", "", "selected alias")
	createSubscriptionCmd.Flags().StringVar(&fromDatabase, "from", "", "database to create the subscription from")
	createSubscriptionCmd.Flags().StringVar(&publicationName, "publication", "", "publication to create the subscription from")
	createSubscriptionCmd.Flags().BoolVar(&apply, "apply", false, "if set, will apply the changes")

	createSubscriptionCmd.MarkPersistentFlagRequired("on") //nolint: errcheck,gosec
	createSubscriptionCmd.MarkFlagRequired("from")         //nolint:errcheck,gosec
	createSubscriptionCmd.MarkFlagRequired("publication")  //nolint:errcheck,gosec

	return createSubscriptionCmd
}
