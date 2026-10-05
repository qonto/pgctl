package cli

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/qonto/pgctl/pkg/pgctl"
	"github.com/spf13/cobra"
)

func grantCmd() *cobra.Command {
	var database, role string
	var apply bool

	grantCmd := &cobra.Command{
		Use:       "grant {" + strings.Join(pgctl.GetSupportedGrants(), "|") + "}",
		Short:     "grant permissions to roles",
		Args:      cobra.ExactArgs(1),
		ValidArgs: pgctl.GetSupportedGrants(),
		PreRun: func(cmd *cobra.Command, args []string) {
			validInput := pgctl.GetSupportedGrants()
			if !slices.Contains(validInput, args[0]) {
				fmt.Println("input validation error in arg[0]: supported grants are", validInput)
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			app.GrantGrant(args[0], database, role, apply)
		},
	}

	grantCmd.Flags().StringVar(&database, "on", "", "selected alias")
	grantCmd.Flags().StringVar(&role, "role", "", "role to grant to")
	grantCmd.Flags().BoolVar(&apply, "apply", false, "if set, will apply the changes")
	grantCmd.MarkFlagRequired("on")   //nolint: errcheck,gosec
	grantCmd.MarkFlagRequired("role") //nolint: errcheck,gosec

	return grantCmd
}
