package pgctl

import (
	"fmt"
	"os"
	"slices"
)

func (a *App) CheckRolesBetweenSourceAndTarget(sourceAlias string, targetAlias string) {
	sourceDB := a.getDatabaseFromAlias(sourceAlias)
	targetDB := a.getDatabaseFromAlias(targetAlias)

	sourceRoles, err := sourceDB.ListRoles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Unable to get roles from source instance: %v\n", err)
		os.Exit(1)
	}

	targetRoles, err := targetDB.ListRoles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Unable to get roles from target instance: %v\n", err)
		os.Exit(1)
	}

	var missingRoles []string
	for _, sourceRole := range sourceRoles {
		if !slices.Contains(targetRoles, sourceRole) {
			missingRoles = append(missingRoles, sourceRole)
		}
	}

	if len(missingRoles) > 0 {
		fmt.Printf("⚠️  The following roles exist on the source instance %s but missing on the target instance %s:\n", sourceAlias, targetAlias)
		for _, role := range missingRoles {
			fmt.Printf("- %s\n", role)
		}
	} else {
		fmt.Printf("✅ All roles of the source instance %s exist on the target instance %s\n", sourceAlias, targetAlias)
	}
}
