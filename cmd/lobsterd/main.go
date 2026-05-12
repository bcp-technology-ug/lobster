package main

import (
	"os"

	"github.com/bcp-technology/lobster/internal/daemon"
)

func main() {
	if err := daemon.NewRootCommand().Execute(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
