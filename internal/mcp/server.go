// Package mcp implements a Model Context Protocol server for Lobster.
//
// The server exposes:
//
//   - Resources: live step catalog, docs pages, example feature files.
//   - Tools: lobster_steps, lobster_validate, lobster_lint, lobster_plan.
//
// Start via: lobster mcp  (runs on stdio, compatible with all MCP clients)
package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/bcp-technology-ug/lobster/internal/steps"
	"github.com/bcp-technology-ug/lobster/internal/steps/builtin"
)

// NewServer constructs and returns a configured MCP server for Lobster.
// lobsterBin is the path to the lobster executable (typically os.Args[0]).
func NewServer(lobsterBin string) *server.MCPServer {
	s := server.NewMCPServer(
		"lobster",
		"0.1.0",
		server.WithResourceCapabilities(true, false),
		server.WithToolCapabilities(true),
	)

	registerResources(s)
	registerTools(s, lobsterBin)

	return s
}

// Serve starts the MCP server on stdio. It blocks until the client disconnects.
func Serve(s *server.MCPServer) error {
	return server.ServeStdio(s)
}

// ── Resources ──────────────────────────────────────────────────────────────────

func registerResources(s *server.MCPServer) {
	// Static: full step catalog.
	s.AddResource(
		mcpgo.NewResource(
			"lobster://steps",
			"Lobster Step Catalog",
			mcpgo.WithMIMEType("text/markdown"),
			mcpgo.WithResourceDescription("All registered step patterns grouped by category. Read this before writing any .feature file."),
		),
		handleStepsResource,
	)

	// Template: category-filtered step catalog.
	s.AddResourceTemplate(
		mcpgo.NewResourceTemplate(
			"lobster://steps/{category}",
			"Lobster Steps by Category",
			mcpgo.WithTemplateDescription("Step patterns for a specific category (http, shell, fs, service, grpc, vars, wait, assert)."),
		),
		handleStepsCategoryResource,
	)

	// Template: docs pages.
	s.AddResourceTemplate(
		mcpgo.NewResourceTemplate(
			"lobster://docs/{topic}",
			"Lobster Documentation",
			mcpgo.WithTemplateDescription("Lobster documentation page. Topics: getting-started, step-definitions, configuration, concepts, hooks, integrations, architecture, cli-reference."),
		),
		handleDocsResource,
	)
}

func handleStepsResource(
	_ context.Context,
	_ mcpgo.ReadResourceRequest,
) ([]mcpgo.ResourceContents, error) {
	catalog, err := buildStepCatalog("")
	if err != nil {
		return nil, err
	}
	return []mcpgo.ResourceContents{
		mcpgo.TextResourceContents{
			URI:      "lobster://steps",
			MIMEType: "text/markdown",
			Text:     catalog,
		},
	}, nil
}

func handleStepsCategoryResource(
	_ context.Context,
	req mcpgo.ReadResourceRequest,
) ([]mcpgo.ResourceContents, error) {
	// Extract category from the URI: lobster://steps/{category}
	category := strings.TrimPrefix(req.Params.URI, "lobster://steps/")
	catalog, err := buildStepCatalog(category)
	if err != nil {
		return nil, err
	}
	return []mcpgo.ResourceContents{
		mcpgo.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "text/markdown",
			Text:     catalog,
		},
	}, nil
}

func handleDocsResource(
	_ context.Context,
	req mcpgo.ReadResourceRequest,
) ([]mcpgo.ResourceContents, error) {
	topic := strings.TrimPrefix(req.Params.URI, "lobster://docs/")
	// Sanitise: only allow simple filenames, no path traversal.
	topic = filepath.Base(topic)
	if strings.Contains(topic, "..") || strings.Contains(topic, "/") {
		return nil, fmt.Errorf("invalid topic: %q", topic)
	}

	// Attempt to read from the docs/ directory relative to the current working dir.
	docPath := filepath.Join("docs", topic+".md")
	content, err := os.ReadFile(docPath) //nolint:gosec // topic is sanitised above
	if err != nil {
		return nil, fmt.Errorf("doc %q not found (looked at %s): %w", topic, docPath, err)
	}
	return []mcpgo.ResourceContents{
		mcpgo.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "text/markdown",
			Text:     string(content),
		},
	}, nil
}

// ── Tools ──────────────────────────────────────────────────────────────────────

func registerTools(s *server.MCPServer, lobsterBin string) {
	// lobster_steps: return the step catalog (equivalent to lobster steps --format markdown).
	s.AddTool(
		mcpgo.NewTool("lobster_steps",
			mcpgo.WithDescription("Returns all registered Lobster step patterns in markdown format. Call this first before writing any feature file."),
			mcpgo.WithString("filter",
				mcpgo.Description("Optional category filter: http, shell, fs, service, grpc, vars, wait, assert"),
			),
		),
		func(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
			filter, _ := req.GetArguments()["filter"].(string)
			catalog, err := buildStepCatalog(filter)
			if err != nil {
				return mcpgo.NewToolResultError(err.Error()), nil
			}
			return mcpgo.NewToolResultText(catalog), nil
		},
	)

	// lobster_validate: validate feature files.
	s.AddTool(
		mcpgo.NewTool("lobster_validate",
			mcpgo.WithDescription("Validates .feature files for syntax errors. Returns validation results. Always run after writing or modifying feature files."),
			mcpgo.WithString("features",
				mcpgo.Description("Glob pattern for feature files (e.g. features/**/*.feature)"),
				mcpgo.Required(),
			),
		),
		func(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
			featuresGlob, _ := req.GetArguments()["features"].(string)
			out, err := runLobster(lobsterBin, "validate", "--features", featuresGlob, "--format", "json")
			if err != nil {
				// validate exits non-zero on validation errors — that's not a tool error.
				return mcpgo.NewToolResultText(out), nil
			}
			return mcpgo.NewToolResultText(out), nil
		},
	)

	// lobster_lint: lint feature files.
	s.AddTool(
		mcpgo.NewTool("lobster_lint",
			mcpgo.WithDescription("Lints .feature files for style and quality issues. Run after validate."),
			mcpgo.WithString("features",
				mcpgo.Description("Glob pattern for feature files (e.g. features/**/*.feature)"),
				mcpgo.Required(),
			),
		),
		func(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
			featuresGlob, _ := req.GetArguments()["features"].(string)
			out, _ := runLobster(lobsterBin, "lint", "--features", featuresGlob, "--format", "json")
			return mcpgo.NewToolResultText(out), nil
		},
	)

	// lobster_plan: dry-run — list scenarios that would execute.
	s.AddTool(
		mcpgo.NewTool("lobster_plan",
			mcpgo.WithDescription("Generates an execution plan (dry run) listing all scenarios that would be executed. Use to verify selection before running."),
			mcpgo.WithString("features",
				mcpgo.Description("Glob pattern for feature files (e.g. features/**/*.feature)"),
				mcpgo.Required(),
			),
			mcpgo.WithString("tags",
				mcpgo.Description("Optional tag expression to filter scenarios (e.g. '@smoke and not @slow')"),
			),
		),
		func(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
			args := req.GetArguments()
			featuresGlob, _ := args["features"].(string)
			tags, _ := args["tags"].(string)

			cmdArgs := []string{"plan", "--features", featuresGlob, "--format", "json"}
			if tags != "" {
				cmdArgs = append(cmdArgs, "--tags", tags)
			}
			out, _ := runLobster(lobsterBin, cmdArgs...)
			return mcpgo.NewToolResultText(out), nil
		},
	)
}

// ── Helpers ────────────────────────────────────────────────────────────────────

// buildStepCatalog renders the step catalog markdown for the given category
// filter (empty = all categories).
func buildStepCatalog(filter string) (string, error) {
	reg := steps.NewRegistry()
	if err := builtin.Register(reg); err != nil {
		return "", fmt.Errorf("loading built-in steps: %w", err)
	}

	defs := reg.Defs()
	if filter != "" {
		fl := strings.ToLower(filter)
		var filtered []*steps.StepDef
		for _, d := range defs {
			if strings.Contains(strings.ToLower(d.Source), fl) {
				filtered = append(filtered, d)
			}
		}
		defs = filtered
	}

	var buf bytes.Buffer
	renderMarkdown(&buf, defs, filter)
	return buf.String(), nil
}

// runLobster executes the lobster binary with the given arguments and returns
// combined stdout+stderr output. It never returns an error for non-zero exit
// codes (those are surfaced as text output to the agent).
func runLobster(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...) //nolint:gosec // bin comes from os.Args[0]
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	_ = cmd.Run() // non-zero exit is normal for validate/lint/plan
	return out.String(), nil
}
