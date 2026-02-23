package pgctl

import (
	"fmt"
	"os"
)

func (a *App) ListExtensions(alias string, listOnAllDatabases bool) {
	db := a.getDatabaseFromAlias(alias)

	listDatabases := []string{db.Database}
	if listOnAllDatabases {
		fmt.Printf("👉 Will run on all databases on %s\n", alias)
		allDatabases, err := db.GetAllDatabases()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Unable to get all databases: %v\n", err)
			os.Exit(1)
		}
		listDatabases = append(listDatabases, allDatabases...)
	}

	for _, d := range listDatabases {
		extensions, err := db.GetExtensions(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Unable to get extensions on %s: %v\n", d, err)
			os.Exit(1)
		}
		if len(extensions) == 0 {
			fmt.Println("| No extensions found for", d)
		} else {
			fmt.Printf("✅ Found extensions for %s : %s\n", d, extensions)
		}
	}
}

func (a *App) UpdateExtensions(alias string, updateOnAllDatabases bool, runMajorUpdates bool, runApply bool) {
	if !runApply {
		fmt.Println("🚧 DRY RUN MODE ACTIVATED 🚧")
	}

	db := a.getDatabaseFromAlias(alias)

	updateDatabases := []string{db.Database}
	if updateOnAllDatabases {
		fmt.Printf("👉 Will run on all databases on %s\n", alias)
		allDatabases, err := db.GetAllDatabases()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Unable to get all databases: %v\n", err)
			os.Exit(1)
		}
		updateDatabases = append(updateDatabases, allDatabases...)
	}

	errorCount := 0
	for _, d := range updateDatabases {
		fmt.Printf("| Retrieving updatable extensions on %s for %s\n", alias, d)
		extensionsUpdatable, err := db.GetExtensionsUpdatable(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Unable to get extensions to update: %v\n", err)
			os.Exit(1)
		}
		if len(extensionsUpdatable) == 0 {
			fmt.Println("| No extensions to update for", d)
		}
		extensionsToUpdate := make([]string, 0)
		for _, e := range extensionsUpdatable {
			if runMajorUpdates || e.NeedOnlyMinorUpdate {
				extensionsToUpdate = append(extensionsToUpdate, e.Name)
				fmt.Printf("✅ Updatable extension found %s : %s ➚ %s\n", e.Name, e.InstalledVersion, e.DefaultVersion)
			} else {
				fmt.Printf("| Ignored extension because it needs a major update %s : %s ➚ %s\n", e.Name, e.InstalledVersion, e.DefaultVersion)
			}
		}

		if runApply && len(extensionsToUpdate) > 0 {
			fmt.Printf("| Updating on %s for %s: %s\n", alias, d, extensionsToUpdate)
			err := db.UpdateExtensions(d, extensionsToUpdate)
			if err != nil {
				errorCount += 1
				fmt.Printf("❌ Could not update extensions for %s: %v\n", d, err)
			} else {
				fmt.Printf("✅ All extensions updated on %s for %s : %s\n", alias, d, extensionsToUpdate)
			}
		}
	}
	if errorCount > 0 {
		os.Exit(1)
	}
}
