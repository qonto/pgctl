package cli

import (
	"fmt"
	"os"

	"github.com/qonto/pgctl/pkg/pgctl"
	"github.com/spf13/cobra"
)

func checkCmd() *cobra.Command {
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Various check tools.",
	}
	checkCmd.AddCommand(checkTablesHaveProperReplicaIdentityCmd())
	checkCmd.AddCommand(checkUserHasReplicationGrantsCmd())
	checkCmd.AddCommand(checkWalLevelIsLogicalCmd())
	checkCmd.AddCommand(checkDatabaseIsEmptyCmd())
	checkCmd.AddCommand(checkSequenceCmd())
	checkCmd.AddCommand(checkSubscriptionLagCmd())
	checkCmd.AddCommand(checkRolesBetweenSourceAndTargetCmd())

	return checkCmd
}

func checkUserHasReplicationGrantsCmd() *cobra.Command {
	checkUserHasReplicationGrantsCmd := &cobra.Command{
		Use:   "user-has-replication-grants",
		Short: "Check if a user has replication grants",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.CheckUserHasReplicationGrants(database)
		},
	}

	checkUserHasReplicationGrantsCmd.PersistentFlags().StringVar(&database, "on", "", "selected alias")
	checkUserHasReplicationGrantsCmd.MarkPersistentFlagRequired("on") //nolint: errcheck,gosec

	return checkUserHasReplicationGrantsCmd
}

func checkWalLevelIsLogicalCmd() *cobra.Command {
	checkWalLevelIsLogicalCmd := &cobra.Command{
		Use:   "wal-level-is-logical",
		Short: "Check if wal level is logical",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.CheckWalLevelIsLogical(database)
		},
	}

	checkWalLevelIsLogicalCmd.PersistentFlags().StringVar(&database, "on", "", "selected alias")
	checkWalLevelIsLogicalCmd.MarkPersistentFlagRequired("on") //nolint: errcheck,gosec

	return checkWalLevelIsLogicalCmd
}

func checkDatabaseIsEmptyCmd() *cobra.Command {
	checkDatabaseIsEmptyCmd := &cobra.Command{
		Use:   "database-is-empty",
		Short: "Check if a database is empty",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.CheckDatabaseIsEmpty(database)
		},
	}

	checkDatabaseIsEmptyCmd.PersistentFlags().StringVar(&database, "on", "", "selected alias")
	checkDatabaseIsEmptyCmd.MarkPersistentFlagRequired("on") //nolint: errcheck,gosec

	return checkDatabaseIsEmptyCmd
}

func checkSequenceCmd() *cobra.Command {
	var source, target string

	checkSequenceCmd := &cobra.Command{
		Use:   "have-similar-sequences",
		Short: "Checks if sequences in the source database are the same as in the target.",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.CheckSequences(source, target)
		},
	}

	checkSequenceCmd.Flags().StringVar(&source, "from", "", "source database instance")
	checkSequenceCmd.Flags().StringVar(&target, "to", "", "target database instance")
	checkSequenceCmd.MarkFlagRequired("from") //nolint: errcheck,gosec
	checkSequenceCmd.MarkFlagRequired("to")   //nolint: errcheck,gosec

	return checkSequenceCmd
}

func checkSubscriptionLagCmd() *cobra.Command {
	var subscriptionName string

	checkSubscriptionLagCmd := &cobra.Command{
		Use:   "subscription-lag",
		Short: "Check the replication lag for a specific subscription.",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			_, err = app.CheckSubscriptionLag(database, subscriptionName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v", err)
				os.Exit(1)
			}
		},
	}

	checkSubscriptionLagCmd.Flags().StringVar(&subscriptionName, "name", "", "subscription name")
	checkSubscriptionLagCmd.MarkFlagRequired("name") //nolint: errcheck,gosec

	checkSubscriptionLagCmd.PersistentFlags().StringVar(&database, "on", "", "selected alias")
	checkSubscriptionLagCmd.MarkPersistentFlagRequired("on") //nolint: errcheck,gosec

	return checkSubscriptionLagCmd
}

func checkTablesHaveProperReplicaIdentityCmd() *cobra.Command {
	var tables []string
	var runOnAllTables bool

	checkTablesHaveProperReplicaIdentityCmd := &cobra.Command{
		Use:   "tables-have-proper-replica-identity",
		Short: "Check if all tables have proper replica identity (not 'nothing' or 'default' without primary key)",
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

				app.CheckTablesHaveProperReplicaIdentity(database, tables)
				return
			}

			if len(tables) > 0 {
				app.CheckTablesHaveProperReplicaIdentity(database, tables)
				return
			}

			selectedTables, err := app.SelectTables(database, "Select tables to check replica identity")
			if err != nil {
				fmt.Printf("Unable to select tables: %v\n", err)
				os.Exit(1)
			}
			app.CheckTablesHaveProperReplicaIdentity(database, selectedTables)
		},
	}

	checkTablesHaveProperReplicaIdentityCmd.Flags().StringSliceVar(&tables, "tables", []string{}, "tables to check")
	checkTablesHaveProperReplicaIdentityCmd.Flags().BoolVar(&runOnAllTables, "all-tables", false, "if set, will run on all tables on the alias provided")

	checkTablesHaveProperReplicaIdentityCmd.PersistentFlags().StringVar(&database, "on", "", "selected alias")
	checkTablesHaveProperReplicaIdentityCmd.MarkPersistentFlagRequired("on") //nolint: errcheck,gosec

	return checkTablesHaveProperReplicaIdentityCmd
}

func checkRolesBetweenSourceAndTargetCmd() *cobra.Command {
	var sourceAlias, targetAlias string

	checkRolesBetweenSourceAndTargetCmd := &cobra.Command{
		Use:   "roles-between-source-and-target",
		Short: "Check if roles of the source instance exist on the target instance",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}
			app.CheckRolesBetweenSourceAndTarget(sourceAlias, targetAlias)
		},
	}

	checkRolesBetweenSourceAndTargetCmd.Flags().StringVar(&sourceAlias, "from", "", "source database instance")
	checkRolesBetweenSourceAndTargetCmd.Flags().StringVar(&targetAlias, "to", "", "target database instance")
	checkRolesBetweenSourceAndTargetCmd.MarkFlagRequired("from") //nolint: errcheck,gosec
	checkRolesBetweenSourceAndTargetCmd.MarkFlagRequired("to")   //nolint: errcheck,gosec

	return checkRolesBetweenSourceAndTargetCmd
}
