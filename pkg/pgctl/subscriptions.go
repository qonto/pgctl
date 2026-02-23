package pgctl

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/qonto/pgctl/internal/postgres"
)

func (a *App) ListSubscriptions(alias string, listOnAllDatabases bool) {
	db := a.getDatabaseFromAlias(alias)

	listDatabases := []string{}
	if listOnAllDatabases {
		fmt.Printf("👉 Will run on all databases on %s\n", alias)
		allDatabases, err := db.GetAllDatabases()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Unable to get all databases: %v\n", err)
			os.Exit(1)
		}
		listDatabases = append(listDatabases, allDatabases...)
	} else {
		listDatabases = append(listDatabases, db.Database)
	}

	for _, d := range listDatabases {
		subscriptions, err := db.GetSubscriptions(d)
		if err != nil {
			fmt.Printf("❌ Failed to get subscriptions: %v\n", err)
			os.Exit(1)
		}

		if len(subscriptions) == 0 {
			fmt.Printf("❌ No subscriptions found on %s on database %s\n", alias, d)
			continue
		}

		subscriptionsNames := make([]string, len(subscriptions))
		for i, subscription := range subscriptions {
			subscriptionsNames[i] = subscription.SubName
		}

		fmt.Printf("✅ Subscriptions on %s on database %s:\n%s\n", alias, d, strings.Join(subscriptionsNames, "\n"))
	}
}

func (a *App) CheckSubscriptionLag(alias string, subscriptionName string) (int64, error) {
	db := a.getDatabaseFromAlias(alias)

	lag, err := db.GetSubscriptionLag(subscriptionName)
	if err != nil {
		return 1, fmt.Errorf("❌ Unable to get subscription lag: %w\n", err)
	}

	if lag == 0 {
		fmt.Printf("✅ Subscription lag for %s on %s: %d bytes\n", subscriptionName, alias, lag)
		return lag, nil
	}

	fmt.Printf("... Subscription lag for %s on %s: %d bytes\n", subscriptionName, alias, lag)
	return lag, nil
}

func (a *App) CreateSubscription(alias string, fromAlias string, publicationName string, apply bool) {
	db := a.getDatabaseFromAlias(alias)

	fromDb := a.getDatabaseFromAlias(fromAlias)

	subscriptionName := a.getSubscriptionName(alias)

	err := a.createSubscriptionPreChecks(db, fromDb, publicationName)
	if err != nil {
		fmt.Printf("❌ Subscription pre-checks failed: %v\n", err)
		return
	}

	if !apply {
		fmt.Println("🚧 DRY RUN MODE ACTIVATED 🚧")
		fmt.Printf("👉 Would create subscription %s on %s from %s on publication %s\n", subscriptionName, alias, fromAlias, publicationName)
		return
	}

	err = db.CreateSubscription(subscriptionName, fromDb, publicationName)
	if err != nil {
		fmt.Printf("❌ Failed to create subscription: %v\n", err)
		fmt.Printf("👉 Drop publication %s to avoid WAL files overload the source database: %s\n", publicationName, fromAlias)
		fmt.Printf("    pgctl drop publication --on %s --name %s", fromAlias, publicationName)
		os.Exit(1)
	}

	fmt.Printf("✅ Subscription %s created on %s from %s on publication %s\n", subscriptionName, alias, fromAlias, publicationName)
}

func (a *App) getSubscriptionName(alias string) string {
	db := a.getDatabaseFromAlias(alias)
	return fmt.Sprintf("sub_%s", strings.ReplaceAll(db.Database, "-", "_"))
}

func (a *App) createSubscriptionPreChecks(db postgres.DB, fromDb postgres.DB, publicationName string) error {
	// Check that wal level is logical
	walLevel, err := db.GetWalLevel()
	if err != nil {
		return err
	}

	if walLevel != walLevelLogical {
		return fmt.Errorf("wal level is not logical: %s", walLevel)
	}

	// Check that user has replication grants
	hasReplicationGrants, err := db.HasReplicationGrants()
	if err != nil {
		return err
	}
	if !hasReplicationGrants {
		return fmt.Errorf("user does not have replication grants")
	}
	// Check that user has subscription grants
	hasSubscriptionGrants, err := db.HasSubscriptionGrants()
	if err != nil {
		return err
	}
	if !hasSubscriptionGrants {
		return fmt.Errorf("user does not have pg_create_subscription grants")
	}

	// Check that the publication exists
	publications, err := fromDb.GetPublications(fromDb.Database)
	if err != nil {
		return err
	}

	foundPublication := slices.ContainsFunc(publications, func(p postgres.QueryPublication) bool {
		return p.PubName == publicationName
	})

	if !foundPublication {
		return fmt.Errorf("publication %s does not exist in %s", publicationName, fromDb.Database)
	}

	return nil
}

func (a *App) DropSubscription(alias string, subscription string, apply bool) {
	db := a.getDatabaseFromAlias(alias)

	if !apply {
		fmt.Println("🚧 DRY RUN MODE ACTIVATED 🚧")
		fmt.Printf("👉 Subscription %s would be dropped on target database %s in %s\n", subscription, db.Database, alias)
		return
	}

	err := db.DropSubscription(subscription)
	if err != nil {
		fmt.Printf("❌ Failed to drop subscription: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Subscription %s dropped on target database %s in %s\n", subscription, db.Database, alias)
}
