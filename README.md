# agrev

**The code review tool that knows your AI agent was here.**

Reviewing agent-generated code with `git diff` is like auditing a contractor's work using only a receipt. You can see *what* changed, but you're missing *why* decisions were made, what the agent tried and abandoned, where it was uncertain, and what downstream effects the changes have.

`agrev` is a single binary that combines diff review with agent trace analysis, static analysis, and interactive review actions. It reads a git diff and an agent conversation trace, cross-references them, and presents an interactive review session where you can understand, assess, and selectively approve changes.

```
  ┌─────────────────┬──────────────────────────────┬─────────────────────┐
  │ Files            │ main.go  [2 findings]        │ Agent Trace (claude)│
  │                  │                              │                     │
  │ V main.go   +2-1 │ @@ -1,5 +1,6 @@             │ > Planning changes  │
  │ X config.go +0-3 │  package main                │ W Write main.go     │
  │ - util.go   +5+0 │                              │ $ go test ./...     │
  │                  │  func main() {               │ > Verify tests pass │
  │                  │ -  println("hello")          │                     │
  │                  │ +  println("hello world")    │                     │
  │                  │ +  println("goodbye")        │                     │
  │                  │  }                           │                     │
  ├─────────────────┴──────────────────────────────┴─────────────────────┤
  │ File 1/3  Line 1/8  +7 -4  unified  risk:medium  1V 1X 1?  ? help   │
  └──────────────────────────────────────────────────────────────────────┘
```

## Features

- **Interactive TUI** — Vim-style navigation, unified and side-by-side diff views, syntax highlighting
- **Agent trace integration** — Reads Claude Code, Aider, and generic JSONL traces to show *why* each change was made
- **Static analysis** — Six analysis passes flag security-sensitive changes, deleted functions with live callers, new dependencies, schema migrations, anti-patterns, and blast radius
- **Review workflow** — Approve (`a`), reject (`x`), or undo (`u`) per file with auto-advance, then generate a patch from only the approved changes
- **CI-ready** — `agrev check` outputs text, JSON, markdown, or HTML reports with risk-based exit codes
- **HTTP API** — `agrev serve` exposes REST endpoints and a WebSocket for building editor plugins and web UIs
- **Zero config** — Single binary, no runtime dependencies, auto-detects traces

## Installation

### From source

```bash
go install github.com/aezell/agrev/cmd/agrev@latest
```

This puts the binary in your `$GOPATH/bin` (usually `~/go/bin`). Make sure it's in your `$PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Build from repo

```bash
git clone https://github.com/aezell/agrev.git
cd agrev
make build
# Binary at ./bin/agrev
```

### Releases

Download pre-built binaries from [GitHub Releases](https://github.com/aezell/agrev/releases) for Linux and macOS (amd64/arm64).

## Quick start

```bash
# Review the last commit
agrev review

# Review a branch against main
agrev review main...HEAD

# Pipe any diff
git diff main...feature | agrev review -

# Non-interactive analysis
agrev check main...HEAD

# Generate a PR description from agent trace
agrev summary
```

## Usage

### `agrev review`

Open an interactive TUI for reviewing changes.

```bash
agrev review [commit-range] [flags]
```

| Flag | Description |
|------|-------------|
| `-t, --trace <path>` | Path to agent trace file |
| `--no-trace` | Skip trace auto-detection |
| `-C, --context <n>` | Lines of context (default: 3) |
| `--stat` | Print diff stats and exit |
| `-o, --output-patch <path>` | Write approved changes as a patch file |
| `--commit-msg` | Print a suggested commit message |

**Keyboard shortcuts:**

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll down / up |
| `n` / `N` | Next / previous file |
| `]` / `[` | Next / previous hunk |
| `a` | Approve current file |
| `x` | Reject current file |
| `u` | Undo decision |
| `Enter` | Finish review (show summary) |
| `v` | Toggle unified / split view |
| `t` | Toggle agent trace panel |
| `Tab` | Switch focus between diff and trace |
| `?` | Help |
| `q` | Quit |

### `agrev check`

Run analysis and output a structured report. Designed for CI pipelines and pre-commit hooks.

```bash
agrev check [commit-range] [flags]
```

| Flag | Description |
|------|-------------|
| `-f, --format <fmt>` | Output: `text`, `json`, `markdown`, `html` |
| `--skip <passes>` | Skip analysis passes (comma-separated) |

**Exit codes:** `0` = clean, `1` = warnings, `2` = high risk.

**Analysis passes:**

| Pass | What it checks |
|------|---------------|
| `security` | Auth, crypto, SQL, subprocess, env vars, filesystem, network |
| `deps` | New dependencies in go.mod, package.json, Cargo.toml, etc. |
| `deleted` | Deleted functions that still have callers in the codebase |
| `schema` | Database migrations and DDL statements |
| `anti_patterns` | Broad exceptions, commented-out code, TODO/HACK, near-duplicates |
| `blast_radius` | Changed functions with many references across the codebase |

### `agrev summary`

Generate a PR description from an agent's conversation trace.

```bash
agrev summary [flags]
```

Auto-detects Claude Code traces from `~/.claude/projects/`, or specify a path with `--trace`.

### `agrev serve`

Start an HTTP API server for editor integrations and web UIs.

```bash
agrev serve [flags]
```

| Flag | Description |
|------|-------------|
| `-a, --addr` | Listen address (default: `127.0.0.1`) |
| `-p, --port` | Listen port (default: `6142`) |

**REST endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/api/analyze` | Run analysis on a diff |
| `POST` | `/api/parse` | Parse a diff into structured files |
| `POST` | `/api/summary` | Generate summary from trace |
| `GET` | `/api/ws` | WebSocket for interactive review |

**Example:**

```bash
agrev serve &

curl -X POST http://localhost:6142/api/analyze \
  -H 'Content-Type: application/json' \
  -d '{"diff": "'"$(git diff main...HEAD)"'"}'
```

## CI Integration

### GitHub Actions

Use the reusable workflow to run `agrev check` on pull requests:

```yaml
name: Agent Review
on:
  pull_request:
    branches: [main]

jobs:
  agrev:
    uses: aezell/agrev/.github/workflows/agrev-check.yaml@main
    with:
      format: markdown
      post-comment: true
```

This will post analysis results as a PR comment and fail the check if high-risk issues are found.

## Agent trace support

`agrev` auto-detects and parses traces from:

| Agent | Format | Detection |
|-------|--------|-----------|
| **Claude Code** | JSONL | `~/.claude/projects/<encoded-path>/` |
| **Aider** | Markdown | `.aider.chat.history.md` in repo root |
| **Generic** | JSONL | `.agent-trace.jsonl` in repo root |

The trace panel shows the agent's reasoning, file operations, and commands alongside the diff, so you can understand the *intent* behind each change.

## License

MIT
