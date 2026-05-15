# AI Agent Integration

Lobster is designed to work seamlessly with AI coding agents. This page covers how to configure agents (VS Code Copilot, Claude Code, Cursor, Windsurf, OpenCode) to write, validate, and debug lobster feature files effectively.

---

## Quick Start

**Every agent session in a lobster project should start with:**

```sh
lobster steps --format markdown
```

This returns the complete, live catalog of available step patterns — formatted for LLM context windows. Use it as the ground truth before writing any Gherkin.

---

## The Bootstrap Protocol

```sh
# 1. Discover capabilities
lobster steps --format markdown

# 2. Write .feature files using ONLY steps from that output
# (see: docs/step-definitions.md)

# 3. Validate syntax
lobster validate --features 'features/**/*.feature'

# 4. Check style and quality
lobster lint --features 'features/**/*.feature'

# 5. Dry-run: see which scenarios will execute
lobster plan --features 'features/**/*.feature' --tags @mytag

# 6. Execute
lobster run --features 'features/**/*.feature' --tags @mytag
```

Never skip step 3. `lobster validate` catches undefined steps (steps that don't match any registered pattern) before they fail at runtime.

---

## Static Guidance Files

Lobster ships several files that agents pick up automatically when a project is opened:

| File | Read by |
|---|---|
| `AGENTS.md` | Claude Code, OpenCode, Cursor, Windsurf |
| `CLAUDE.md` | Claude-family agents |
| `.github/copilot-instructions.md` | VS Code Copilot (auto-applied to `*.feature` files) |
| `llms.txt` | Any agent/tool that supports the llms.txt standard |

These files all teach the same bootstrap protocol and point agents to `lobster steps` for the authoritative step catalog.

### Scaffold guidance files into a new project

```sh
lobster init --project my-app --with-ai-guidance
```

This adds all agent guidance files and VS Code prompt files to the new project.

### Add to an existing project

Copy from the lobster repo:
- `AGENTS.md`
- `CLAUDE.md`
- `.github/copilot-instructions.md`
- `.github/prompts/` (prompt files)

---

## MCP Server Integration

Lobster includes a built-in [Model Context Protocol](https://modelcontextprotocol.io/) server. When connected, agents can call lobster tools directly in conversation — without switching to a terminal.

### Start the server

```sh
lobster mcp
```

The server runs on stdio and is compatible with all MCP-capable agents.

### VS Code configuration

Add to `.vscode/mcp.json` in your project:

```json
{
  "servers": {
    "lobster": {
      "type": "stdio",
      "command": "lobster",
      "args": ["mcp"]
    }
  }
}
```

Or globally in VS Code user settings (`mcp.servers`).

### Claude Code configuration

Add to `~/.claude/mcp_servers.json`:

```json
{
  "lobster": {
    "command": "lobster",
    "args": ["mcp"]
  }
}
```

Or per-project: `claude mcp add lobster -- lobster mcp`

### Cursor configuration

In Cursor settings → MCP:

```json
{
  "mcpServers": {
    "lobster": {
      "command": "lobster",
      "args": ["mcp"]
    }
  }
}
```

### What the MCP server exposes

**Resources** (read-only data agents can fetch):

| URI | Content |
|---|---|
| `lobster://steps` | Full step catalog (markdown) |
| `lobster://steps/{category}` | Steps filtered by category (http, shell, fs, service, grpc, vars, wait, assert) |
| `lobster://docs/{topic}` | Documentation pages by filename (e.g. `lobster://docs/getting-started`) |

**Tools** (agents can call these like functions):

| Tool | Description |
|---|---|
| `lobster_steps` | Returns step catalog; optional `filter` parameter |
| `lobster_validate` | Validates feature files; returns JSON results |
| `lobster_lint` | Lints feature files for style issues; returns JSON results |
| `lobster_plan` | Dry-run execution plan; accepts `features` and optional `tags` |

---

## Workflow Prompt Files

Lobster provides pre-built VS Code prompt files (`.prompt.md`) for the most common agent tasks. These work in VS Code Copilot (via `@workspace /scaffold-feature`) and any agent that supports the prompt file standard.

| File | Purpose |
|---|---|
| `.github/prompts/scaffold-feature.prompt.md` | Create a new feature test end-to-end |
| `.github/prompts/debug-scenario.prompt.md` | Diagnose and fix a failing scenario |
| `.github/prompts/add-auth-to-tests.prompt.md` | Add bearer/basic/API-key authentication |
| `.github/prompts/migrate-to-custom-step.prompt.md` | Extract repeated steps into a custom step |

Scaffold these into a project with:

```sh
lobster init --with-ai-guidance
```

---

## `lobster steps` Output Formats

| Format | Best for |
|---|---|
| `text` (default) | Terminal reading, quick reference |
| `markdown` | Pasting into agent context, documentation |
| `json` | Programmatic parsing, tooling integration |

```sh
lobster steps                          # grouped text
lobster steps --format markdown        # LLM-optimised context block
lobster steps --format json            # machine-readable array
lobster steps --filter http            # HTTP category only
lobster steps --filter http --format markdown  # combined
```

---

## Step Pattern Syntax

Lobster uses Go regular expressions internally for step matching, but `lobster steps` outputs human-readable patterns:

| Display | Matches |
|---|---|
| `<string>` | Any quoted string argument |
| `<n>` | Any integer |
| `<n>s` | Timeout in seconds (e.g. `30s`) |
| `GET\|POST\|...` | Any listed HTTP method |

Variable interpolation (`${VAR_NAME}`) is supported in any quoted argument and is resolved at step execution time.

---

## Common Pitfalls for Agents

| Mistake | Consequence | Fix |
|---|---|---|
| Inventing step text | `ErrUndefined` at runtime | Run `lobster steps` first; only use listed patterns |
| Skipping `lobster validate` | Undefined steps discovered too late | Always validate after writing |
| Hardcoding secrets in feature files | Security exposure | Use `${VAR_NAME}` + `lobster run --env KEY=value` |
| Using `--keep-stack` in CI | Container leak | Only use `--keep-stack` for local debugging |
| Multiple `Feature:` blocks in one file | Parse error | One feature per file |
| Calling `lobster run` without a `lobster.yaml` | Config error | Run `lobster init` first |

---

## Integration Checklist

- [ ] `lobster` is installed and on `$PATH` (`lobster --version`)
- [ ] `lobster.yaml` exists in the project root
- [ ] Agent guidance files are present (`AGENTS.md`, `.github/copilot-instructions.md`)
- [ ] For MCP: `.vscode/mcp.json` (VS Code) or equivalent agent config is set up
- [ ] Agent has been instructed to run `lobster steps --format markdown` before writing
