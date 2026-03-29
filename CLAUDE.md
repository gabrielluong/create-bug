# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
go build -o create-bug .              # compile CLI → ./create-bug
go build -o create-bug-tui ./cmd/tui/ # compile TUI → ./create-bug-tui
go run .                              # run CLI without building
go run ./cmd/tui/                     # run TUI without building
```

No test runner is configured yet.

## Architecture

Go CLI tool and Bubbletea TUI for creating Bugzilla bugs via the REST API. Designed for standalone use and as a Claude Code SKILL.

```
main.go              → calls cmd.Execute() (CLI entry point)
cmd/
  root.go            → root cobra command (is the create-bug command itself)
  create_bug.go      → flag parsing, default merging, fuzzy resolution, output, history display
  tui/
    main.go          → TUI entry point: loads config, starts tea.NewProgram with alt screen
internal/bugzilla/
  create_bug.go      → CreateBug() business logic entry point
  components.go      → hardcoded component lists (known products) + fuzzy resolution
internal/client/
  client.go          → HTTP transport: Client, CreateBugParams, error codes
internal/config/
  config.go          → Config, BugDefaults, Load() (env > file > defaults), ConfigDir()
internal/history/
  history.go         → Load(), Append(), Clear() for ~/.config/create-bug/history.json
internal/tui/
  app.go             → Root AppModel, screen routing, help bar
  theme.go           → Hex color palette (violet/slate scheme)
  keys.go            → KeyMap with all global bindings
  form/
    model.go         → FormModel: all fields, focus management, submission, lifecycle states
  component/
    fuzzyselect.go   → Single-value fuzzy dropdown (wraps textinput + sahilm/fuzzy)
    bugidinput.go    → Multi-value bug ID input with history autocomplete
internal/update/
  update.go          → LatestVersion(), CheckCached(), RefreshCache(), Run(), IsNewer()
```

**Key conventions — CLI:**
- `internal/client/` owns all HTTP logic. New API calls go here.
- The root command *is* the create-bug command — no subcommands.
- `--json` flag for machine-readable stdout output (Claude SKILL integration).
- Errors to **stderr** via `fmt.Fprintf(os.Stderr, ...)`. `RunE` returns the error to trigger `os.Exit(1)`.
- `SilenceErrors` and `SilenceUsage` on root — Cobra won't double-print.
- Bugzilla returns `{ "error": true, "code": N, "message": "..." }` — check this **before** HTTP status.
- Requires `config.APIKey`; exit early if empty before any network call.
- Use `cmd.Flags().Changed("flag")` to distinguish "not passed" from "passed empty" when merging with config defaults.
- `--component` (`-c`) supports fuzzy matching via `sahilm/fuzzy` for known products. Unknown products pass through for API validation.
- Short flags: `-p` (product), `-c` (component), `-b` (blocks), `-d` (depends-on), `-H` (history).
- Tab completion registered for `--component` flag (returns known components when `--product` is set or defaulted from config).
- Tab completion registered for `--blocks` and `--depends-on` flags: `[meta]` bugs ranked first, then newest-first; each entry displays as `Bug {id} - {summary} [Product :: Component]`; handles comma-separated multi-value input.
- `--history` (`-H`) shows recently filed bugs; `--clear-history` removes the history file. Both short-circuit before any bug creation logic.
- Filing history stored in `~/.config/create-bug/history.json`, capped to `historySize` (default 20, configurable in config file).
- History silently fails on write errors (fire-and-forget) so it never blocks bug creation.
- Version is a package-level constant in `cmd/root.go` (`const version`), used by both cobra and the update checker.
- `PersistentPreRun` reads the update cache (fast, local file) and spawns `update.RefreshCache()` as a goroutine (async, non-blocking). `PersistentPostRun` prints the update notice to stderr if one was set.
- Update cache stored in `~/.config/create-bug/update_check.json`; refreshed at most once per 24h. Version source: GitHub tags API.
- `--update` fetches latest version synchronously and runs `go install github.com/gabrielluong/create-bug@latest` if outdated.

**Key conventions — TUI:**
- `internal/tui/form/model.go` owns all form state and submission. Focus zones: Summary → Component → Description → Blocks → Depends on → Submit.
- Up/Down arrows navigate between fields unless a dropdown is open or Description is focused (textarea captures arrows for line movement).
- `component.FuzzySelect`: single-value selector. Clears input on focus to show full list; auto-confirms best match on blur; restores previous selection if user clears input without choosing.
- `component.BugIDInput`: multi-value comma-separated input. Fuzzy-matches against current segment (text after last comma); excludes already-entered IDs; `[meta]` bugs ranked first then newest-first; selecting from dropdown appends `id, ` and re-opens for next entry.
- `bugzilla.ProductComponents` is the single source of truth for component lists — both CLI completion and TUI fuzzy selector read from it. Priority components listed first in the slice.
- Component ordering: prioritized items (Homepage, Top Sites, Stories, Toolbar, Tabs) at top, then alphabetical.
- History autocomplete ranking (both CLI and TUI): bugs with `[meta]` in summary first, then `CreatedAt` descending.
- On successful submit, history is appended immediately so the new bug appears in Blocks/Depends on autocomplete when filing another.
- TUI uses hex colors (`#7C3AED` violet family, `#1E293B` slate surfaces) — keep the palette in `internal/tui/theme.go`.

## Auth

Set `BUGZILLA_API_KEY` (and optionally `BUGZILLA_URL`) as environment variables, or write them to `~/.config/create-bug/config.json`:

```json
{
  "apiKey": "your-key",
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
