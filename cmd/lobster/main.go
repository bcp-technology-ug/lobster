package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/bcp-technology/lobster/internal/cli"
	"github.com/bcp-technology/lobster/internal/ui"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		var exitErr *cli.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		_, _ = fmt.Fprint(os.Stderr, ui.RenderError("Error", err.Error(), "", ""))
		os.Exit(1)
	}
}
