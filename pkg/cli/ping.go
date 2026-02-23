package cli

import (
	"fmt"
	"os"

	"github.com/qonto/pgctl/pkg/pgctl"
	"github.com/spf13/cobra"
)

func pingCmd() *cobra.Command {
	pingCmd := &cobra.Command{
		Use:   "ping",
		Short: "Validate connectivity with configured databases.",
		Run: func(cmd *cobra.Command, args []string) {
			app, err := pgctl.New()
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			err = app.Ping(args)
			if err != nil {
				fmt.Printf("%v\n", err)
				os.Exit(1)
			}
		},
	}

	return pingCmd
}
