package cli

import (
	"fmt"
	"os"

	"github.com/qonto/pgctl/pkg/pgctl"
	"github.com/spf13/cobra"
)

func dropCmd() *cobra.Command {
	dropCmd := &cobra.Command{
		Use:   "drop",
		Short: "Various drop tools.",
	}
	dropCmd.AddCommand(dropPublicationCmd())
	dropCmd.AddCommand(dropSubscriptionCmd())

	dropCmd.PersistentFlags().StringVar(&database, "on", "", "selected alias")
	dropCmd.MarkPersistentFlagRequired("on") //nolint: errcheck,gosec

	return dropCmd
}

func dropPublicationCmd() *cobra.Command {
	var publication string
	var apply bool

	dropPublicationCmd := &cobra.Command{
		Use:   "publication",
		Short: "Drop a publication",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.DropPublication(database, publication, apply)
		},
	}

	dropPublicationCmd.PersistentFlags().StringVar(&publication, "name", "", "publication to drop")
	dropPublicationCmd.Flags().BoolVar(&apply, "apply", false, "if set, will apply the changes")
	dropPublicationCmd.MarkPersistentFlagRequired("name") //nolint: errcheck,gosec

	return dropPublicationCmd
}

func dropSubscriptionCmd() *cobra.Command {
	var subscription string
	var apply bool

	dropSubscriptionCmd := &cobra.Command{
		Use:   "subscription",
		Short: "Drop a subscription",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.DropSubscription(database, subscription, apply)
		},
	}

	dropSubscriptionCmd.PersistentFlags().StringVar(&subscription, "name", "", "subscription to drop")
	dropSubscriptionCmd.Flags().BoolVar(&apply, "apply", false, "if set, will apply the changes")

	dropSubscriptionCmd.MarkPersistentFlagRequired("name") //nolint: errcheck,gosec

	return dropSubscriptionCmd
}
