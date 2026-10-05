package cli

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/qonto/pgctl/pkg/pgctl"
	"github.com/spf13/cobra"
)

func revokeCmd() *cobra.Command {
	var database, role string
	var apply bool

	revokeCmd := &cobra.Command{
		Use:       "revoke {" + strings.Join(pgctl.GetSupportedGrants(), "|") + "}",
		Short:     "revoke some grants",
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

			app.RevokeGrant(args[0], database, role, apply)
		},
	}

	revokeCmd.Flags().StringVar(&database, "on", "", "selected alias")
	revokeCmd.Flags().StringVar(&role, "role", "", "role to revoke grant from")
	revokeCmd.Flags().BoolVar(&apply, "apply", false, "if set, will apply the changes")
	revokeCmd.MarkFlagRequired("on")   //nolint: errcheck,gosec
	revokeCmd.MarkFlagRequired("role") //nolint: errcheck,gosec

	return revokeCmd
}
