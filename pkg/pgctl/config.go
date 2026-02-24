package pgctl

import (
	"fmt"
	"os"

	"github.com/qonto/pgctl/internal/config"
	"github.com/qonto/pgctl/internal/postgres"
)

type App struct {
	Config *config.Config
}

func New() (*App, error) {
	c, err := config.Load()
	if err != nil {
		return nil, err
	}
	return &App{Config: c}, err
}

func (a *App) ShowConfig() {
	for name, db := range a.Config.Databases {
		fmt.Printf("%s:\n", name)
		fmt.Printf("  host: %s\n", db.Host)
		fmt.Printf("  port: %d\n", db.Port)
		fmt.Printf("  database: %s\n", db.Database)
		fmt.Printf("  role: %s\n", db.Role)
		fmt.Println()
	}
}

func (a *App) Ping(aliases []string) error {
	var err error

	if len(aliases) == 0 {
		aliases, err = a.askUserAliasesToPing()
		if err != nil {
			return fmt.Errorf("❌ Failed to get database aliases from user input: %w", err)
		}
	}

	for _, alias := range aliases {
		db := a.getDatabaseFromAlias(alias)

		if err := db.Ping(); err != nil {
			return fmt.Errorf("❌ Could not establish connection for alias %s: %w", alias, err)
		}

		fmt.Printf("✅ Successful connection for alias %s\n", alias)
	}
	return nil
}

func (a *App) getDatabaseFromAlias(alias string) postgres.DB {
	db, ok := a.Config.Databases[alias]

	if !ok {
		fmt.Printf("❌ Alias %s does not exist. Add it to your configuration file and retry.\n", alias)
		os.Exit(1)
	}
	return db
}
