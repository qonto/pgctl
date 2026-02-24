package pgctl

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/qonto/pgctl/internal/postgres"
)

func (a *App) ListTables(alias string, addSchemaPrefix bool) []string {
	db := a.getDatabaseFromAlias(alias)

	var tablesList []string
	if addSchemaPrefix {
		tablesWithSchema, err := db.GetTablesWithSchema()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Unable to get all tables with their schema: %v\n", err)
			os.Exit(1)
		}
		tablesList = postgres.ConvertToStringArray(tablesWithSchema)
	} else {
		tables, err := db.GetTables()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Unable to get all tables: %v\n", err)
			os.Exit(1)
		}
		tablesList = postgres.ConvertToStringArray(tables)
	}
	fmt.Println(getStringSeparatedByCommas(tablesList))
	return tablesList
}

func (a *App) CheckTablesHaveProperReplicaIdentity(alias string, tables []string) {
	if len(tables) == 0 {
		fmt.Printf("❌ No tables provided. Add them to the command and retry.\n")
		os.Exit(1)
	}
	db := a.getDatabaseFromAlias(alias)

	tablesWithoutProperReplicaIdentity, err := db.GetTablesWithoutProperReplicaIdentity(tables)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Unable to check if tables have proper replica identity: %v\n", err)
		os.Exit(1)
	}

	if len(tablesWithoutProperReplicaIdentity) > 0 {
		fmt.Printf("❌ Tables do not have proper replica identity: %s\n", tablesWithoutProperReplicaIdentity)
		os.Exit(1)
	}
	fmt.Printf("✅ All tables have proper replica identity\n")
}

func (a *App) GetTables(alias string) ([]string, error) {
	db := a.getDatabaseFromAlias(alias)

	tables, err := db.GetTables()
	if err != nil {
		return nil, err
	}

	tablesNames := postgres.ConvertToStringArray(tables)

	return tablesNames, nil
}

func (a *App) validateTableNames(tables []string, databaseTables []string) error {
	// Check for duplicates
	tableMap := make(map[string]bool)
	var duplicates []string
	for _, table := range tables {
		if tableMap[table] {
			duplicates = append(duplicates, table)
		} else {
			tableMap[table] = true
		}
	}

	if len(duplicates) > 0 {
		return fmt.Errorf("table %s is listed multiple times", strings.Join(duplicates, ", "))
	}

	// Check that all tables exist in databaseTables
	for _, table := range tables {
		if !slices.Contains(databaseTables, table) {
			return fmt.Errorf("table %s does not exist", table)
		}
	}

	return nil
}
