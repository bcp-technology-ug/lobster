package main

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/bcp-technology-ug/lobster/internal/daemon"
	"github.com/bcp-technology-ug/lobster/internal/log"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "failed to initialise logger:", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()
	ctx := log.WithLogger(context.Background(), logger)
	if err := daemon.NewRootCommand().ExecuteContext(ctx); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
