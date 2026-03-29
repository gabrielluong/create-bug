# create-bug

A CLI tool for creating bugs on [Bugzilla](https://www.bugzilla.org) from the terminal. Supports fuzzy component matching, config-based defaults, and JSON output for use as a [Claude Code](https://claude.ai/code) SKILL.

## Install

```sh
go install github.com/gabrielluong/create-bug@latest
```

Or build from source:

```sh
git clone https://github.com/gabrielluong/create-bug.git
cd create-bug
go build -o create-bug .
```

## Usage

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

# JSON output
create-bug "Crash on startup" --json
```

## Configuration

Create `~/.config/create-bug/config.json`:

```json
{
  "apiKey": "your-bugzilla-api-key",
  "baseUrl": "https://your-bugzilla-instance.example.com",
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

## Shell Completion

```sh
# zsh
create-bug completion zsh > "${fpath[1]}/_create-bug"

# bash
create-bug completion bash > /etc/bash_completion.d/create-bug

# fish
create-bug completion fish > ~/.config/fish/completions/create-bug.fish
```

## License

[MIT](LICENSE)
