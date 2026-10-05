package pgctl

import (
	"context"
	"fmt"
	"slices"
	"time"
)

func (a *App) InitRelocation(sourceAlias string, targetAlias string, apply bool) error {
	// CHECKS
	err := a.Ping([]string{sourceAlias, targetAlias})
	if err != nil {
		return err
	}
	// on source:
	a.CheckUserHasReplicationGrants(sourceAlias)
	a.CheckWalLevelIsLogical(sourceAlias)
	// on target:
	a.CheckUserHasReplicationGrants(targetAlias)
	a.CheckUserHasSubscriptionGrants(targetAlias)
	a.CheckWalLevelIsLogical(targetAlias)
	a.CheckDatabaseIsEmpty(targetAlias)

	// CREATE
	allTables := a.ListTables(sourceAlias, false)

	// Filter out known-safe unlogged tables, fail on unknown ones
	filteredTables, err := a.filterUnloggedTables(sourceAlias, allTables)
	if err != nil {
		return err
	}

	a.CopySchema(sourceAlias, targetAlias, true, apply)
	publicationName := a.CreatePublication(sourceAlias, filteredTables, apply)
	a.CreateSubscription(targetAlias, sourceAlias, publicationName, apply)

	if apply {
		fmt.Println("✅ Relocation successfully initialized")
	}
	if !apply {
		fmt.Println("🚀 To apply the changes, run the command with --apply flag")
	}
	return nil
}

func (a *App) filterUnloggedTables(alias string, tables []string) ([]string, error) {
	db := a.getDatabaseFromAlias(alias)

	unloggedTables, err := db.GetUnloggedTables()
	if err != nil {
		return nil, fmt.Errorf("unable to detect unlogged tables: %w", err)
	}

	if len(unloggedTables) == 0 {
		return tables, nil
	} else {
		fmt.Printf("Unlogged tables detected and skipped...")
	}

	var filtered []string
	for _, t := range tables {
		if !slices.Contains(unloggedTables, t) {
			filtered = append(filtered, t)
		}
	}

	return filtered, nil
}

func (a *App) RunRelocation(sourceAlias, targetAlias string, apply bool) error {
	// CHECKS
	err := a.Ping([]string{sourceAlias, targetAlias})
	if err != nil {
		return err
	}
	ready, err := a.CheckSubscriptionReady(sourceAlias)
	if err != nil {
		return err
	}
	if !ready {
		return fmt.Errorf("❌ Precheck failed: subscription not ready, because at least one table initial copy is not complete")
	}

	// LAG WAIT
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	err = a.waitForZeroSubscriptionLag(ctx, 500*time.Millisecond, sourceAlias, targetAlias)
	if err != nil {
		return err
	}

	// SEQUENCES
	a.CopySequences(sourceAlias, targetAlias, apply)
	if apply {
		a.CheckSequences(sourceAlias, targetAlias)
	}

	// DROP PUBSUB
	a.DropPublication(sourceAlias, a.getPublicationName(sourceAlias), apply)
	a.DropSubscription(targetAlias, a.getSubscriptionName(targetAlias), apply)

	if apply {
		fmt.Println("✅ Relocation successfully ended")
		fmt.Printf("\n👉 You may update connection strings and reconnect clients to %s\n", targetAlias)
	} else {
		fmt.Printf("\n🚀 To apply the changes, run the command with --apply flag\n")
	}

	return nil
}

func (a *App) waitForZeroSubscriptionLag(ctx context.Context, poll time.Duration, sourceAlias, targetAlias string) error {
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for {
		lag, err := a.CheckSubscriptionLag(sourceAlias, a.getSubscriptionName(targetAlias))
		if err != nil {
			return err
		}
		if lag == 0 {
			return nil
		}
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
