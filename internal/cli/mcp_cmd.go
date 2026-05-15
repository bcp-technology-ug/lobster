package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	lobstermcp "github.com/bcp-technology-ug/lobster/internal/mcp"
)

func newMCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start a Model Context Protocol server (stdio)",
		Long: `Start a lobster MCP server on stdio for use with AI coding agents.

Connect any MCP-compatible agent (VS Code Copilot, Claude Code, Cursor, Windsurf)
to this server by adding lobster as an MCP server in the agent's configuration.

The server exposes:
  Resources:
    lobster://steps              — full step catalog (markdown)
    lobster://steps/{category}   — filtered by category
    lobster://docs/{topic}       — documentation pages

  Tools:
    lobster_steps                — return step catalog, with optional filter
    lobster_validate             — validate .feature files
    lobster_lint                 — lint .feature files
    lobster_plan                 — dry-run execution plan

Example VS Code mcp.json entry:
  {
    "servers": {
      "lobster": {
        "type": "stdio",
        "command": "lobster",
        "args": ["mcp"]
      }
    }
  }`,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			bin, err := os.Executable()
			if err != nil {
				bin = "lobster"
			}
			srv := lobstermcp.NewServer(bin)
			if err := lobstermcp.Serve(srv); err != nil {
				return fmt.Errorf("mcp server: %w", err)
			}
			return nil
		},
	}
	return cmd
}
