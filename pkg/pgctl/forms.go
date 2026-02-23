package pgctl

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/qonto/pgctl/internal/postgres"
)

func (a *App) SelectTables(alias string, title string) ([]string, error) {
	db := a.getDatabaseFromAlias(alias)

	tablesFrom, err := db.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	selectedTables := []string{}
	tableNames := postgres.ConvertToStringArray(tablesFrom)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(title).
				Options(huh.NewOptions(tableNames...)...).
				Value(&selectedTables),
		),
	).WithTheme(huh.ThemeBase16())

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("failed to run form: %w", err)
	}

	return selectedTables, nil
}

func (a *App) askUserAliasesToPing() ([]string, error) {
	var selected []string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("").
				OptionsFunc(func() []huh.Option[string] {
					return huh.NewOptions(a.Config.Aliases()...)
				}, nil).
				Value(&selected),
		),
	).WithTheme(huh.ThemeBase16())

	err := form.Run()
	if err != nil {
		return nil, err
	}

	return selected, nil
}
