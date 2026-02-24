package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Var to be able to update it with goreleaser with ldflags
var DefaultVersion = "dev" //nolint

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display current version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(DefaultVersion)
		},
	}
}
