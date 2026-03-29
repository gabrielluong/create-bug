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

# JSON output
create-bug "Crash on startup" --json

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
| `--json` | | Output raw JSON |
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

## Updates

The tool checks for new versions once per 24 hours in the background and prints a notice to stderr after any command if a newer version is available:

```
A new version (v0.5.0) is available. Run: go install github.com/gabrielluong/create-bug@latest
```

To update immediately:

```sh
create-bug --update
```

## License

[MIT](LICENSE)
