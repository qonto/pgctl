package pgctl

import (
	"fmt"
	"os"

	"github.com/qonto/pgctl/internal/postgres"
)

const (
	walLevelLogical = "logical"
)

func (a *App) ListDatabases(alias string) {
	db := a.getDatabaseFromAlias(alias)

	allDatabases, err := db.GetAllDatabases()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Unable to get all databases: %v\n", err)
		os.Exit(1)
	}

	for _, d := range allDatabases {
		fmt.Println(d)
	}
}

func (a *App) CheckWalLevelIsLogical(alias string) {
	db := a.getDatabaseFromAlias(alias)

	walLevel, err := db.GetWalLevel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Unable to get wal level: %v\n", err)
		os.Exit(1)
	}

	if walLevel == walLevelLogical {
		fmt.Printf("✅ Wal level is logical: %s\n", walLevel)
	} else {
		fmt.Printf("❌ Wal level is not logical: %s\n", walLevel)
		os.Exit(1)
	}
}

func (a *App) CheckDatabaseIsEmpty(alias string) {
	db := a.getDatabaseFromAlias(alias)

	empty, err := db.DatabaseIsEmpty()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Unable to check if database is empty: %v\n", err)
		os.Exit(1)
	}

	if empty {
		fmt.Printf("✅ Database %s in %s is empty\n", db.Database, alias)
	} else {
		fmt.Printf("❌ Database %s in %s is not empty\n", db.Database, alias)
		fmt.Println("👉 Try running \"DROP EXTENSION pg_stat_statements;\"")
		os.Exit(1)
	}
}

func (a *App) checkDatabaseVersionCompatibility(sourceDB, targetDB postgres.DB) error {
	// Get source database version
	sourceVersion, err := sourceDB.GetServerVersion()
	if err != nil {
		return fmt.Errorf("unable to get source database version: %w", err)
	}

	// Get target database version
	targetVersion, err := targetDB.GetServerVersion()
	if err != nil {
		return fmt.Errorf("unable to get target database version: %w", err)
	}

	// Check that major versions match
	if sourceVersion.Major != targetVersion.Major {
		return fmt.Errorf("source database major version %d does not match target database major version %d",
			sourceVersion.Major, targetVersion.Major)
	}

	return nil
}
