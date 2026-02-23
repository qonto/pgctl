package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/qonto/pgctl/internal/config"
	"github.com/qonto/pgctl/internal/postgres"
	"github.com/qonto/pgctl/pkg/pgctl"
	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the configuration.",
	}

	configCmd.AddCommand(configInitCmd())
	configCmd.AddCommand(configShowCmd())

	return configCmd
}

func configInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a new configuration file.",
		Run: func(cmd *cobra.Command, args []string) {
			err := explainUserPGCTLConfiguration()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			databases := make(map[string]postgres.DB)

			for {
				configName, database, err := promptForDBConnectionDetails()
				if err != nil {
					fmt.Printf("err: %v\n", err)
					os.Exit(1)
				}

				databases[configName] = database

				addMore, err := promptForMoreDBs()
				if err != nil {
					fmt.Printf("err: %v\n", err)
					os.Exit(1)
				}

				if !addMore {
					break
				}
			}

			cfg := &config.Config{
				Databases: databases,
			}

			err = config.Write(cfg)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("✅ Configuration file written successfully!\n👉 Use `pgctl config show` command to view its content.")
		},
	}
}

func explainUserPGCTLConfiguration() error {
	note := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("pgctl - config initialization").
				Description(`
Get started with pgctl:

👉 Choose an alias for each database you intend to interact with. This name will be the unique reference in every following pgctl command!
👉 Fill out the connection parameters in the form.

pgctl will write for you the corresponding .pgctl.yaml file with your configuration in the current directory.
				`),
		),
	).WithTheme(huh.ThemeBase16())

	err := note.Run()
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return err
}

func promptForDBConnectionDetails() (string, postgres.DB, error) {
	var (
		alias    string
		host     string
		port     string
		database string
		role     string
		password string
	)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Alias").
				Value(&alias).
				Suggestions([]string{"target"}).
				Validate(isNotEmpty),
			huh.NewInput().
				Title("Host").
				Value(&host).
				Suggestions([]string{"myhost.hello.co"}).
				Validate(isValidHost),
			huh.NewInput().
				Title("Port").
				Value(&port).
				Suggestions([]string{"5432"}).
				Validate(isValidPort),
			huh.NewInput().
				Title("Database").
				Value(&database).
				Suggestions([]string{"mydb"}).
				Validate(isNotEmpty),
			huh.NewInput().
				Title("Role").
				Value(&role).
				Suggestions([]string{"myrole"}).
				Validate(isNotEmpty),
			huh.NewInput().
				Title("Password").
				Value(&password).
				EchoMode(huh.EchoModePassword).
				Validate(isNotEmpty),
		),
	).WithTheme(huh.ThemeBase16())

	err := form.Run()
	if err != nil {
		return "", postgres.DB{}, err
	}

	portNum, _ := strconv.Atoi(port)
	db := postgres.DB{
		Host:     host,
		Port:     portNum,
		Database: database,
		Role:     role,
		Password: password,
	}

	return alias, db, nil
}

func promptForMoreDBs() (bool, error) {
	var addMore bool

	continueForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Add another database?").
				Value(&addMore),
		),
	).WithTheme(huh.ThemeBase16())

	err := continueForm.Run()
	if err != nil {
		return false, err
	}

	return addMore, nil
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display the current configuration.",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.ShowConfig()
		},
	}
}
