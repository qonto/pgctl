package pgctl

import (
	"fmt"
	"os"
)

func (a *App) CheckUserHasReplicationGrants(alias string) {
	db := a.getDatabaseFromAlias(alias)

	hasReplicationGrants, err := db.HasReplicationGrants()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Unable to get replication grants: %v\n", err)
		os.Exit(1)
	}

	if hasReplicationGrants {
		fmt.Printf("✅ User %s has replication grants\n", db.Role)
	} else {
		fmt.Printf("❌ User %s does not have replication grants on %s, give them with `ALTER ROLE %[1]s REPLICATION` or grant them a replication role\n", db.Role, alias)
		os.Exit(1)
	}
}

func (a *App) CheckUserHasSubscriptionGrants(alias string) {
	db := a.getDatabaseFromAlias(alias)

	hasSubscriptionGrants, err := db.HasSubscriptionGrants()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Unable to get subscription grants: %v\n", err)
		os.Exit(1)
	}

	if hasSubscriptionGrants {
		fmt.Printf("✅ User %s has subscription grants\n", alias)
	} else {
		fmt.Printf("❌ User %s does not have subscription grants on %s, give them with `GRANT pg_create_subscription TO %[1]s`\n", db.Role, alias)
		os.Exit(1)
	}
}
