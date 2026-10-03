package main

import (
	"errors"
	"fmt"
	"github.com/tasnimzotder/portman/internal/cli"
	"os"
)

func main() {
	if err := cli.RootCmd.Execute(); err != nil {
		code := 1
		var exit *cli.ExitError
		if errors.As(err, &exit) {
			code = exit.Code
		}
		if exit == nil || !exit.Silent {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(code)
	}
}
