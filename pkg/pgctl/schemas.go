package pgctl

import (
	"fmt"
	"os"

	"github.com/qonto/pgctl/internal/postgres"
)

func (a *App) CopySchema(sourceAlias, targetAlias string, allTables bool, runApply bool) {
	if !runApply {
		fmt.Println("🚧 DRY RUN MODE ACTIVATED 🚧")
	}
	if !allTables {
		fmt.Println("❌ Not selecting all tables is not supported yet. Please add --all-tables flag to your command.")
		os.Exit(1)
	}

	// 1. Validate aliases exist
	sourceDB := a.getDatabaseFromAlias(sourceAlias)
	targetDB := a.getDatabaseFromAlias(targetAlias)

	// 2. Pre-flight checks
	if err := a.copySchemaPreChecks(sourceDB, targetDB); err != nil {
		fmt.Printf("❌ Pre-flight check failed: %v\n", err)
		os.Exit(1)
	}

	// 3. Execute schema copy
	if runApply {
		err := a.executeSchemaCopy(sourceDB, targetDB)
		if err != nil {
			fmt.Printf("❌ Schema copy failed: %v\n", err)
			fmt.Printf("👉 The target database %s might be in a defective state. After solving whatever issue occurred, and before retrying this command, drop it and recreate it.", targetAlias)
			os.Exit(1)
		}
		fmt.Printf("✅ Successfully copied schema from %s to %s\n", sourceAlias, targetAlias)
	} else {
		fmt.Printf("👉 Would have copied schema from %s to %s\n", sourceAlias, targetAlias)
	}
}

func (a *App) copySchemaPreChecks(sourceDB, targetDB postgres.DB) error {
	// Check that target database is empty
	empty, err := targetDB.DatabaseIsEmpty()
	if err != nil {
		return fmt.Errorf("unable to check if target database is empty: %w", err)
	}

	if !empty {
		return fmt.Errorf("target database is not empty")
	}

	// Check that source and target databases have the same major version
	// If that's not the case, this operation is less safe
	if err := a.checkDatabaseVersionCompatibility(sourceDB, targetDB); err != nil {
		fmt.Println("⚠️ Database version compatibility check failed:", err)
	}

	// Check pg_dump compatibility
	if err := targetDB.CheckPgDumpCompatibility(); err != nil {
		return fmt.Errorf("pg_dump compatibility check failed: %w", err)
	}

	return nil
}

func (a *App) executeSchemaCopy(sourceDB, targetDB postgres.DB) error {
	// Dump schema from source database
	schemaSQL, err := sourceDB.DumpSchema()
	if err != nil {
		return fmt.Errorf("failed to dump schema: %w", err)
	}

	// Restore schema on target database
	err = targetDB.RestoreSchema(schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to restore schema: %w", err)
	}

	return nil
}
