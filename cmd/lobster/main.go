package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/bcp-technology-ug/lobster/internal/cli"
	"github.com/bcp-technology-ug/lobster/internal/log"
	"github.com/bcp-technology-ug/lobster/internal/ui"
)

func main() {
	logger, err := zap.NewDevelopment(zap.WithCaller(false))
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "failed to initialise logger:", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()
	ctx := log.WithLogger(context.Background(), logger)
	if err := cli.NewRootCommand().ExecuteContext(ctx); err != nil {
		var exitErr *cli.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		_, _ = fmt.Fprint(os.Stderr, ui.RenderError("Error", err.Error(), "", ""))
		os.Exit(1)
	}
}
