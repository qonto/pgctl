package pgctl

import (
	"context"
	"fmt"
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
	a.CopySchema(sourceAlias, targetAlias, true, apply)
	publicationName := a.CreatePublication(sourceAlias, allTables, apply)
	a.CreateSubscription(targetAlias, sourceAlias, publicationName, apply)

	if apply {
		fmt.Println("✅ Relocation successfully initialized")
	}
	if !apply {
		fmt.Println("🚀 To apply the changes, run the command with --apply flag")
	}
	return nil
}

func (a *App) RunRelocation(sourceAlias, targetAlias string, apply bool) error {
	// CHECKS
	err := a.Ping([]string{sourceAlias, targetAlias})
	if err != nil {
		return err
	}
	// LAG WAIT
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	err = a.waitForZeroSubscriptionLag(ctx, 15*time.Second, sourceAlias, targetAlias)
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
		select {
		case <-ticker.C:
			if lag == 0 {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
