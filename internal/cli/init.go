package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bcp-technology-ug/lobster/internal/ui"
)

func newInitCommand(_ *viper.Viper) *cobra.Command {
	var (
		project       string
		workspace     string
		features      string
		compose       string
		noInteractive bool
		force         bool
		withAIGuide   bool
	)

	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Scaffold a new lobster test project",
		Long:  "Create lobster.yaml, .lobster/ directory, and a sample feature file in the target path.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.LogoSmall())

			root := "."
			if len(args) > 0 {
				root = args[0]
			}
			root = filepath.Clean(root)

			lobsterYAML := filepath.Join(root, "lobster.yaml")
			if _, err := os.Stat(lobsterYAML); err == nil && !force {
				return fmt.Errorf("lobster.yaml already exists at %s — use --force to overwrite or --config to point to an existing config", lobsterYAML)
			}

			// Collect inputs: interactive form, stdin fallback, or flags.
			if !noInteractive && !cmd.Flags().Changed("project") {
				fields := &ui.InitFields{
					Project:   project,
					Workspace: workspace,
					Features:  features,
					Compose:   compose,
				}
				// Apply defaults before showing the form / reading stdin.
				if fields.Features == "" {
					fields.Features = "features/**/*.feature"
				}

				if ui.IsInteractive() {
					form := ui.NewInitForm(fields)
					if err := form.Run(); err != nil {
						if ui.IsFormAborted(err) {
							return fmt.Errorf("init cancelled")
						}
						return fmt.Errorf("form: %w", err)
					}
				} else {
					// Non-TTY stdin fallback: read up to 4 newline-delimited lines.
					// Usage: printf 'my-project\n\n\n\n' | lobster init .
					if err := ui.ReadInitFieldsFromReader(cmd.InOrStdin(), fields); err != nil {
						return err
					}
				}
				project = fields.Project
				workspace = fields.Workspace
				features = fields.Features
				compose = fields.Compose
			} else if noInteractive || cmd.Flags().Changed("project") {
				// --no-interactive or explicit --project flag: project is required.
				if strings.TrimSpace(project) == "" {
					_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Error", "--project is required in non-interactive mode", "", ""))
					return &ExitError{Code: ExitConfigError}
				}
			}

			// Apply defaults for omitted optional fields.
			if strings.TrimSpace(features) == "" {
				features = "features/**/*.feature"
			}

			// 1. Create project root if it doesn't exist.
			if err := os.MkdirAll(root, 0o755); err != nil {
				return fmt.Errorf("create project directory: %w", err)
			}

			// 2. Create .lobster/ directory.
			lobsterDir := filepath.Join(root, ".lobster")
			if err := os.MkdirAll(lobsterDir, 0o755); err != nil {
				return fmt.Errorf("create .lobster directory: %w", err)
			}

			// 3. Create features/ directory.
			featuresDir := filepath.Join(root, "features")
			if err := os.MkdirAll(featuresDir, 0o755); err != nil {
				return fmt.Errorf("create features directory: %w", err)
			}

			// 4. Write lobster.yaml.
			if err := os.WriteFile(lobsterYAML, []byte(buildLobsterYAML(project, workspace, features, compose)), 0o644); err != nil {
				return fmt.Errorf("write lobster.yaml: %w", err)
			}

			// 5. Write sample feature file.
			exampleFeature := filepath.Join(featuresDir, "example.feature")
			if _, statErr := os.Stat(exampleFeature); os.IsNotExist(statErr) {
				if err := os.WriteFile(exampleFeature, []byte(sampleFeatureFile(project)), 0o644); err != nil {
					return fmt.Errorf("write example feature: %w", err)
				}
			}

			// 6. Optionally scaffold AI agent guidance files.
			if withAIGuide {
				if err := scaffoldAIGuidance(root, project); err != nil {
					return fmt.Errorf("scaffold AI guidance: %w", err)
				}
			}

			// 7. Success output.
			tree := fmt.Sprintf(
				"%s/\n  %s\n  %s/\n  features/\n    example.feature",
				filepath.Base(root), "lobster.yaml", ".lobster",
			)
			if withAIGuide {
				tree += "\n  AGENTS.md\n  CLAUDE.md\n  .github/\n    copilot-instructions.md\n    prompts/\n  .vscode/\n    mcp.json"
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderSuccess(
				"Project initialised",
				"Created:\n\n"+ui.StyleCode.Render(tree),
			))

			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name (required in non-interactive mode)")
	cmd.Flags().StringVar(&workspace, "workspace", "", "default workspace (leave blank for root)")
	cmd.Flags().StringVar(&features, "features", "", "feature file glob (default: features/**/*.feature)")
	cmd.Flags().StringVar(&compose, "compose", "", "docker-compose file path (optional)")
	cmd.Flags().BoolVar(&noInteractive, "no-interactive", false, "skip interactive form and use flags only")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing lobster.yaml if present")
	cmd.Flags().BoolVar(&withAIGuide, "with-ai-guidance", false, "scaffold AI agent guidance files (AGENTS.md, CLAUDE.md, .github/copilot-instructions.md, .github/prompts/)")

	return cmd
}

// buildLobsterYAML returns the minimal lobster.yaml content.
func buildLobsterYAML(project, workspace, features, compose string) string {
	var b strings.Builder

	b.WriteString("# lobster.yaml — generated by `lobster init`\n")
	b.WriteString("# Full reference: https://github.com/bcp-technology-ug/lobster/blob/main/docs/configuration.md\n\n")

	b.WriteString("project: " + yamlString(project) + "\n\n")

	b.WriteString("workspace:\n")
	if workspace != "" {
		b.WriteString("  selected: " + yamlString(workspace) + "\n")
	} else {
		b.WriteString("  # selected: my-workspace  # uncomment for monorepo isolation\n")
	}
	b.WriteString("\n")

	b.WriteString("features:\n")
	b.WriteString("  paths:\n")
	b.WriteString("    - " + yamlString(features) + "\n\n")

	b.WriteString("compose:\n")
	if compose != "" {
		b.WriteString("  files:\n")
		b.WriteString("    - " + yamlString(compose) + "\n")
	} else {
		b.WriteString("  # files:\n")
		b.WriteString("  #   - docker-compose.yaml\n")
	}
	b.WriteString("  migrations:\n")
	b.WriteString("    mode: auto\n\n")

	b.WriteString("persistence:\n")
	b.WriteString("  sqlite:\n")
	b.WriteString("    path: .lobster/lobster.db\n")

	return b.String()
}

// sampleFeatureFile returns a minimal .feature file to help users get started.
func sampleFeatureFile(project string) string {
	name := project
	if name == "" {
		name = "my project"
	}
	return fmt.Sprintf(`Feature: %s health check
  Verify that core services are reachable before running full scenarios.

  Scenario: API responds to health check
    Given the service "api" is running
    When I send a GET request to "/healthz"
    Then the response status should be 200
`, name)
}

// yamlString quotes a string value if it contains characters that need quoting.
func yamlString(s string) string {
	if s == "" {
		return `""`
	}
	needsQuote := strings.ContainsAny(s, ": #{}[]|>&*!,'\"\\")
	if needsQuote {
		return fmt.Sprintf("%q", s)
	}
	return s
}

// scaffoldAIGuidance writes agent guidance files into a lobster project root.
// Existing files are not overwritten.
func scaffoldAIGuidance(root, project string) error {
	// .github/ directories for Copilot and prompts.
	githubDir := filepath.Join(root, ".github")
	promptsDir := filepath.Join(githubDir, "prompts")
	vscodeDir := filepath.Join(root, ".vscode")
	for _, dir := range []string{githubDir, promptsDir, vscodeDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	type file struct {
		path    string
		content string
	}

	files := []file{
		{
			path:    filepath.Join(root, "AGENTS.md"),
			content: agentsMarkdown(project),
		},
		{
			path:    filepath.Join(root, "CLAUDE.md"),
			content: claudeMarkdown(project),
		},
		{
			path:    filepath.Join(githubDir, "copilot-instructions.md"),
			content: copilotInstructionsMarkdown(),
		},
		{
			path:    filepath.Join(vscodeDir, "mcp.json"),
			content: vscodeMCPJSON(),
		},
		{
			path:    filepath.Join(promptsDir, "scaffold-feature.prompt.md"),
			content: scaffoldFeaturePrompt(),
		},
		{
			path:    filepath.Join(promptsDir, "debug-scenario.prompt.md"),
			content: debugScenarioPrompt(),
		},
	}

	for _, f := range files {
		if _, err := os.Stat(f.path); err == nil {
			continue // do not overwrite existing files
		}
		if err := os.WriteFile(f.path, []byte(f.content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", f.path, err)
		}
	}
	return nil
}

func agentsMarkdown(project string) string {
	name := project
	if name == "" {
		name = "this project"
	}
	return fmt.Sprintf(`# Lobster — AI Agent Guide (%s)

Lobster is a CLI-first BDD end-to-end test runner. Read the full guide at:
https://github.com/bcp-technology-ug/lobster/blob/main/AGENTS.md

## Bootstrap: Do This First

`+"```"+`sh
lobster steps --format markdown   # discover all available step patterns
`+"```"+`

**Only use step patterns from that output.** Never invent step text.

## The 6-Step Workflow

`+"```"+`sh
lobster steps --format markdown                               # 1. discover
# 2. write .feature files using ONLY those patterns
lobster validate --features 'features/**/*.feature'           # 3. syntax check
lobster lint     --features 'features/**/*.feature'           # 4. quality check
lobster plan     --features 'features/**/*.feature'           # 5. dry run
lobster run      --features 'features/**/*.feature'           # 6. execute
`+"```"+`

## Rules
- One `+"`Feature:`"+` per .feature file
- Tags: `+"`@smoke`"+`, `+"`@e2e`"+`, `+"`@integration`"+`, `+"`@quarantine`"+`
- Variable interpolation: `+"`${VAR_NAME}`"+` in any quoted step argument
- DocString steps end with `+"`:`"+` — body goes in indented `+"`\"\"\"`"+` blocks
- Run `+"`lobster validate`"+` after every edit
`, name)
}

func claudeMarkdown(project string) string {
	name := project
	if name == "" {
		name = "this project"
	}
	return fmt.Sprintf(`# CLAUDE.md — %s (Lobster project)

This is a lobster BDD test project. See AGENTS.md for the full agent guide.

## Critical rules

1. Always run `+"`lobster steps --format markdown`"+` before writing feature files.
2. Workflow: write → `+"`lobster validate`"+` → `+"`lobster lint`"+` → `+"`lobster plan`"+` → `+"`lobster run`"+`.
3. Step text must match a registered pattern exactly.
4. Use `+"`${VAR_NAME}`"+` for variable interpolation in quoted arguments.
5. DocString steps end with `+"`:`"+` followed by `+"`\"\"\"`"+` indented blocks.
`, name)
}

func copilotInstructionsMarkdown() string {
	return `---
applyTo: "**/*.feature"
---
# Lobster — VS Code Copilot Instructions

Run ` + "`lobster steps --format markdown`" + ` to get the full step catalog before suggesting completions.
Only suggest step patterns from that output. Never invent step text.

## Workflow
` + "```" + `sh
lobster steps --format markdown        # discover steps
lobster validate --features 'features/**/*.feature'  # validate after writing
lobster lint     --features 'features/**/*.feature'  # quality check
lobster plan     --features 'features/**/*.feature'  # dry run
lobster run      --features 'features/**/*.feature'  # execute
` + "```" + `

## Feature file conventions
- One ` + "`Feature:`" + ` per file; ` + "`Background:`" + ` for shared setup
- Tags: ` + "`@smoke`" + `, ` + "`@e2e`" + `, ` + "`@integration`" + `, ` + "`@quarantine`" + `
- Variable interpolation: ` + "`${VAR_NAME}`" + ` in any quoted argument
- DocString steps end with ` + "`:`" + ` followed by ` + "`\"\"\"`" + ` block
- Do NOT skip ` + "`lobster validate`" + ` after edits
`
}

func scaffoldFeaturePrompt() string {
	return `---
mode: agent
description: Scaffold a new Gherkin feature file for this lobster project
---

Before writing any Gherkin, run:
` + "```sh\nlobster steps --format markdown\n```" + `

Then write the feature file using ONLY the step patterns from that output.
After writing, run:
` + "```sh\nlobster validate --features 'features/**/*.feature'\nlobster lint     --features 'features/**/*.feature'\n```" + `
`
}

func debugScenarioPrompt() string {
	return `---
mode: agent
description: Debug a failing lobster scenario
---

If you see ` + "`ErrUndefined`" + `:
` + "```sh\nlobster steps --format markdown  # compare with failing step text\n```" + `

For environment issues:
` + "```sh\nlobster doctor\n```" + `

Isolate the failure:
` + "```sh\nlobster run --features 'path/to/file.feature' --scenario-regex \"Scenario name\" --keep-stack -vv\n```" + `
`
}

func vscodeMCPJSON() string {
	return `{
  "servers": {
    "lobster": {
      "type": "stdio",
      "command": "lobster",
      "args": ["mcp"]
    }
  }
}
`
}
