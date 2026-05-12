package main

import (
	"os"

	"github.com/bcp-technology/lobster/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
