package main

import (
	"os"

	"github.com/tasnimzotder/portman/internal/cli"
)

func main() {
	if err := cli.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
