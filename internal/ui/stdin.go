package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ReadInitFieldsFromReader populates dst by reading up to four newline-delimited
// lines from r. The expected order matches the interactive Huh form:
//
//  1. Project name  (required — returns an error if empty after reading)
//  2. Workspace     (optional — empty line keeps existing default)
//  3. Features glob (optional — empty line keeps existing default)
//  4. Compose file  (optional — empty line keeps existing default)
//
// This path is used when stdin is not a TTY (e.g. in a pipe) and
// --no-interactive was not passed. It allows the init command to be driven
// from a shell script or a lobster feature file step without requiring a PTY.
func ReadInitFieldsFromReader(r io.Reader, dst *InitFields) error {
	scanner := bufio.NewScanner(r)

	fields := []*string{
		&dst.Project,
		&dst.Workspace,
		&dst.Features,
		&dst.Compose,
	}

	for _, field := range fields {
		if !scanner.Scan() {
			break
		}
		line := strings.TrimRight(scanner.Text(), "\r")
		if line != "" {
			*field = line
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading init fields from stdin: %w", err)
	}

	if strings.TrimSpace(dst.Project) == "" {
		return fmt.Errorf("project name is required (first line must be non-empty)")
	}
	return nil
}
