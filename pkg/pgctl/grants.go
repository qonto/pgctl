package pgctl

import (
	"fmt"
	"os"
)

func GetSupportedGrants() []string {
	return []string{"connect"}
}

func (a *App) GrantGrant(grant, alias, role string, apply bool) {
	db := a.getDatabaseFromAlias(alias)
	if !apply {
		fmt.Println("🚧 DRY RUN MODE ACTIVATED 🚧")
		fmt.Printf("👉 Would have grant %s in %s (%s) to %s\n", grant, db.Database, alias, role)
		return
	}

	err := db.GrantGrant(grant, role)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to grant: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ %s granted to %s in %s (%s)\n", grant, role, db.Database, alias)
}

func (a *App) RevokeGrant(grant, alias, role string, apply bool) {
	db := a.getDatabaseFromAlias(alias)
	if !apply {
		fmt.Println("🚧 DRY RUN MODE ACTIVATED 🚧")
		fmt.Printf("👉 Would have revoked %s in %s (%s) from %s\n", grant, db.Database, alias, role)
		return
	}

	err := db.RevokeGrant(grant, role)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to revoke grant: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ %s revoked from %s in %s (%s)\n", grant, role, db.Database, alias)
}
