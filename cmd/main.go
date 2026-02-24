package main

import (
	"fmt"
	"os"

	"github.com/qonto/pgctl/pkg/cli"
)

func main() {
	rootCmd := cli.RootCmd()

	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
