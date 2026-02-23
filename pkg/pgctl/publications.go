package pgctl

import (
	"fmt"
	"os"
	"strings"

	"github.com/qonto/pgctl/internal/postgres"
)

func (a *App) ListPublications(alias string, listOnAllDatabases bool) {
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
		publications, err := db.GetPublications(d)
		if err != nil {
			fmt.Printf("❌ Failed to get publications: %v\n", err)
			os.Exit(1)
		}

		if len(publications) == 0 {
			fmt.Printf("❌ No publications found on %s on database %s\n", alias, d)
			continue
		}

		publicationsNames := make([]string, len(publications))
		for i, publication := range publications {
			publicationsNames[i] = publication.PubName
		}

		fmt.Printf("✅ Publications on %s on database %s:\n%s\n", alias, d, strings.Join(publicationsNames, "\n"))
	}
}

func (a *App) CreatePublication(alias string, tables []string, apply bool) string {
	db := a.getDatabaseFromAlias(alias)

	err := a.createPublicationPreChecks(db, tables)
	if err != nil {
		fmt.Printf("❌ Publication pre-checks failed: %v\n", err)
		os.Exit(1)
	}

	publicationName := a.getPublicationName(alias)

	if !apply {
		fmt.Println("🚧 DRY RUN MODE ACTIVATED 🚧")
		fmt.Printf("👉 Would create publication %s on %s on tables:\n%s\n", publicationName, alias, strings.Join(tables, "\n"))
		return publicationName
	}

	err = db.CreatePublication(publicationName, tables)
	if err != nil {
		fmt.Printf("❌ Failed to create publication: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Publication %s created on %s on tables:\n%s\n", publicationName, alias, strings.Join(tables, "\n"))
	return publicationName
}

func (a *App) createPublicationPreChecks(db postgres.DB, tables []string) error {
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

	// Validate the tables input (duplicates, existence)
	actualTables, err := db.GetTables()
	if err != nil {
		return err
	}

	actualTablesNames := postgres.ConvertToStringArray(actualTables)

	err = a.validateTableNames(tables, actualTablesNames)
	if err != nil {
		return err
	}

	// Check that all tables have proper replica identity
	tablesWithoutProperReplicaIdentity, err := db.GetTablesWithoutProperReplicaIdentity(tables)
	if err != nil {
		return err
	}

	if len(tablesWithoutProperReplicaIdentity) > 0 {
		return fmt.Errorf("tables do not have proper replica identity (either set to 'nothing' or set to 'default' with no primary key): %s", tablesWithoutProperReplicaIdentity)
	}

	return nil
}

func (a *App) getPublicationName(alias string) string {
	db := a.getDatabaseFromAlias(alias)
	return fmt.Sprintf("pub_%s", strings.ReplaceAll(db.Database, "-", "_"))
}

func (a *App) DropPublication(alias string, publication string, apply bool) {
	db := a.getDatabaseFromAlias(alias)

	if !apply {
		fmt.Println("🚧 DRY RUN MODE ACTIVATED 🚧")
		fmt.Printf("👉 Publication %s would be dropped on source database %s in %s\n", publication, db.Database, alias)
		return
	}

	err := db.DropPublication(publication)
	if err != nil {
		fmt.Printf("❌ Failed to drop publication: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Publication %s dropped on source database %s in %s\n", publication, db.Database, alias)
}
