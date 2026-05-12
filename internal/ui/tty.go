package ui

import (
	"os"

	"github.com/charmbracelet/x/term"
)

// IsInteractive reports whether stdin is a terminal (interactive TTY).
// Use this to decide whether to run Huh forms or fall back to flags.
func IsInteractive() bool {
	return term.IsTerminal(os.Stdin.Fd())
}
