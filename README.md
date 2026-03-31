# create-bug

A CLI tool and interactive TUI for creating bugs on [Bugzilla](https://www.bugzilla.org) from the terminal. Supports fuzzy component matching, config-based defaults, and JSON output for use as a [Claude Code](https://claude.ai/code) SKILL.

## Install

```sh
go install github.com/gabrielluong/create-bug@latest
```

Both binaries are included. Build from source:

```sh
git clone https://github.com/gabrielluong/create-bug.git
cd create-bug
go build -o create-bug .            # CLI
go build -o create-bug-tui ./cmd/tui/  # TUI
```

## TUI

Launch the interactive form:

```sh
create-bug-tui
```

### Fields

| Field | Notes |
|-------|-------|
| Summary | Required |
| Component | Fuzzy-filtered dropdown for known products; free-text otherwise |
| Description | Inline textarea; supports multi-line editing |
| Blocks | Comma-separated bug IDs; autocomplete from filing history |
| Depends on | Comma-separated bug IDs; autocomplete from filing history |
| Whiteboard | Free-text whiteboard value |
| Keywords | Comma-separated keywords |

### Navigation

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle between fields |
| `↑` / `↓` | Move between fields (or navigate an open dropdown) |
| `Enter` | Confirm dropdown selection |
| `Esc` | Close dropdown |
| `Ctrl+S` | Submit bug |
| `Ctrl+N` | File another bug (after submit) |
| `Ctrl+C` | Quit |

### Autocomplete ranking

In the Blocks and Depends on fields, bugs with `[meta]` in their summary are ranked first in the autocomplete list, followed by most-recently-filed bugs. The same ranking applies to CLI tab completion.

## CLI

```sh
# With all flags
create-bug "Crash on startup" \
  -p "Product" -c "Component" \
  --version unspecified --type defect

# With config defaults (only summary needed)
create-bug "Crash on startup"

# Fuzzy component matching
create-bug "Theme issue" --component "design"
# stderr: Matched component: "Design System and Theming" (from "design")

# Block and depend on other bugs
create-bug "Follow-up fix" --blocks 123,456 --depends-on 789

# JSON output (includes id, url, summary)
create-bug "Crash on startup" --json

# Dry run — print resolved params without filing
create-bug "Crash on startup" --dry-run

# View recently filed bugs
create-bug --history

# View as JSON
create-bug --history --json
```

## Configuration

Create `~/.config/create-bug/config.json`:

```json
{
  "apiKey": "your-bugzilla-api-key",
  "baseUrl": "https://your-bugzilla-instance.example.com",
  "historySize": 20,
  "defaults": {
    "product": "Product",
    "component": "General",
    "version": "unspecified",
    "type": "defect",
    "priority": "P3",
    "severity": "normal",
    "platform": "All",
    "os": "All"
  }
}
```

`historySize` controls how many recently filed bugs to remember (default: 20). History is saved to `~/.config/create-bug/history.json`.

### Whiteboard and keyword suggestions

Built-in suggestions for `--whiteboard` and `--keywords` are defined in `internal/suggestions/whiteboard.json` and `internal/suggestions/keywords.json` and embedded into the binary at build time.

To extend them with your own values, create one or both of these files:

```json
// ~/.config/create-bug/whiteboard.json
["[my-custom-tag]", "[team-flag]"]

// ~/.config/create-bug/keywords.json
["my-keyword", "team-specific"]
```

User entries are appended after the built-ins (deduped). Omitting a file leaves the built-in list unchanged. To change the built-ins themselves, edit the JSON files in `internal/suggestions/` and rebuild.

Environment variables take priority over the config file:

- `BUGZILLA_API_KEY` — API key for authentication (required)
- `BUGZILLA_URL` — Base URL (required if not set in config)

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--product` | `-p` | Product the bug is filed against |
| `--component` | `-c` | Component (supports fuzzy matching for known products) |
| `--summary` | | Bug summary (or pass as positional argument) |
| `--version` | | Product version |
| `--type` | | Bug type: `defect`, `enhancement`, `task` |
| `--description` | | Initial comment / bug description |
| `--priority` | | P1-P5 |
| `--severity` | | S1-S4, enhancement, normal |
| `--platform` | | Hardware platform |
| `--os` | | Operating system |
| `--assigned-to` | | Assignee email |
| `--cc` | | Comma-separated CC emails |
| `--blocks` | `-b` | Comma-separated bug IDs this blocks |
| `--depends-on` | `-d` | Comma-separated bug IDs this depends on |
| `--alias` | | Short alias |
| `--status` | | Initial status |
| `--whiteboard` | `-w` | Whiteboard field value |
| `--keywords` | `-k` | Comma-separated keywords |
| `--json` | | Output raw JSON (`id`, `url`, `summary`) |
| `--dry-run` | | Print resolved params as JSON without filing |
| `--history` | `-H` | Show recently filed bugs |
| `--clear-history` | | Clear the filing history |
| `--update` | | Update to the latest version |

## Shell Completion

```sh
# zsh
create-bug completion zsh > "${fpath[1]}/_create-bug"

# bash
create-bug completion bash > /etc/bash_completion.d/create-bug

# fish
create-bug completion fish > ~/.config/fish/completions/create-bug.fish
```

Tab completion is supported for:
- `--component` — known components for the selected product
- `--blocks` / `--depends-on` — bug IDs from your filing history, displayed as `Bug {id} - {summary} [Product :: Component]`; supports comma-separated multi-value input; `[meta]` bugs ranked first
- `--whiteboard` — built-in and user-configured whiteboard values
- `--keywords` — built-in and user-configured keywords; supports comma-separated multi-value input

## Updates

The tool checks for new versions once per 24 hours in the background and prints a notice to stderr after any command if a newer version is available:

```
A new version (v0.5.0) is available. Run: go install github.com/gabrielluong/create-bug@latest
```

To update immediately:

```sh
create-bug --update
```

## Claude Code Skill

This repo includes a `/create-bugs` skill for [Claude Code](https://claude.ai/code) that converts work into a set of Bugzilla bugs with dependency chains. It works from a plan file, inline text, or the current conversation — no plan file required.

### Setup

Add this repo as a local Claude Code plugin to make the `/create-bugs` skill available.

### Usage

```sh
# From a plan file
/create-bugs path/to/PLAN.md

# From inline text
/create-bugs implement OAuth token refresh with silent renewal and logout on expiry

# From the current conversation (tasks discussed, work completed, decisions made)
/create-bugs
```

Claude will:

1. Determine the source (file, inline text, or conversation context)
2. Check filing history to identify and skip already-filed bugs
3. Ask if there is a tracking/metabug the bugs should block against
4. Show the full breakdown — including skipped items — with dependency annotations for review
5. Wait for confirmation before filing anything
6. File bugs one at a time via `create-bug --json`, using `--depends-on` only where tasks are truly sequential and `--blocks <metabug_id>` on every bug if a metabug was provided
7. Report each filed bug as `Bug <id> - <summary>` and print a final summary

## License

[MIT](LICENSE)
