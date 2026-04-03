# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
go build -o create-bug .              # compile CLI → ./create-bug
go build -o create-bug-tui ./cmd/create-bug-tui/ # compile TUI → ./create-bug-tui
go run .                                         # run CLI without building
go run ./cmd/create-bug-tui/                     # run TUI without building
```

No test runner is configured yet.

## Architecture

Go CLI tool and Bubbletea TUI for creating Bugzilla bugs via the REST API. Designed for standalone use and as a Claude Code SKILL.

```
main.go              → calls cmd.Execute() (CLI entry point)
cmd/
  root.go            → root cobra command (is the create-bug command itself)
  create_bug.go      → flag parsing, default merging, fuzzy resolution, output, history display
  create-bug-tui/
    main.go          → TUI entry point: loads config, starts tea.NewProgram with alt screen
internal/bugzilla/
  create_bug.go      → CreateBug() business logic entry point
  components.go      → fuzzy resolution against cached component list
  componentcache.go  → GetCachedComponents(), SetCachedComponents(), EnsureComponents() — persistent cache at ~/.config/create-bug/components.json
internal/client/
  client.go          → HTTP transport: Client, CreateBugParams, GetProductComponents(), error codes
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
    bugidinput.go       → Multi-value bug ID input with history autocomplete
    multistringinput.go → Multi-value string input with fuzzy dropdown (used for keywords)
internal/suggestions/
  suggestions.go     → LoadWhiteboard(), LoadKeywords() — embeds built-in JSON files, merges with ~/.config/create-bug/{whiteboard,keywords}.json
  whiteboard.json    → embedded built-in whiteboard suggestions (edit and rebuild to change defaults)
  keywords.json      → embedded built-in keyword suggestions (edit and rebuild to change defaults)
internal/update/
  update.go          → LatestVersion(), CheckCached(), RefreshCache(), Run(), IsNewer()
```

**Key conventions — CLI:**
- `internal/client/` owns all HTTP logic. New API calls go here.
- The root command *is* the create-bug command — no subcommands.
- `--json` flag for machine-readable stdout output (Claude SKILL integration). Returns `{"id", "url", "summary"}`.
- `--dry-run` flag: runs all validation and fuzzy resolution, then prints resolved `CreateBugParams` as `{"dry_run": true, "params": {...}}` without making any API call or writing history.
- Errors to **stderr** via `fmt.Fprintf(os.Stderr, ...)`. `RunE` returns the error to trigger `os.Exit(1)`.
- `SilenceErrors` and `SilenceUsage` on root — Cobra won't double-print.
- Bugzilla returns `{ "error": true, "code": N, "message": "..." }` — check this **before** HTTP status.
- Requires `config.APIKey`; exit early if empty before any network call.
- Use `cmd.Flags().Changed("flag")` to distinguish "not passed" from "passed empty" when merging with config defaults.
- `--component` (`-c`) supports fuzzy matching via `sahilm/fuzzy` against the cached component list for the product. `EnsureComponents()` is called before `ResolveComponent()` to populate the cache on first use. Unknown products (not in cache) pass through for API validation.
- Component list is fetched live from the Bugzilla API (`GET /rest/product?names=…`) and cached at `~/.config/create-bug/components.json`. The cache persists across invocations; there is no TTL — it is populated on first use and reused thereafter.
- Short flags: `-p` (product), `-c` (component), `-b` (blocks), `-d` (depends-on), `-w` (whiteboard), `-k` (keywords), `-H` (history).
- Tab completion registered for `--component` flag: reads from the component cache via `GetCachedComponents()` when `--product` is set or defaulted from config. Returns no completions if the product is not yet cached.
- Tab completion registered for `--blocks` and `--depends-on` flags: `[meta]` bugs ranked first, then newest-first; each entry displays as `Bug {id} - {summary} [Product :: Component]`; handles comma-separated multi-value input.
- Tab completion registered for `--whiteboard` (single value) and `--keywords` (comma-separated multi-value); both source from `suggestions.Load*()`.
- `internal/suggestions` is the single source of truth for whiteboard/keyword lists — built-in defaults merged with user JSON files in config dir.
- `--history` (`-H`) shows recently filed bugs; `--clear-history` removes the history file. Both short-circuit before any bug creation logic.
- Filing history stored in `~/.config/create-bug/history.json`, capped to `historySize` (default 20, configurable in config file).
- History silently fails on write errors (fire-and-forget) so it never blocks bug creation.
- Version is a package-level constant in `cmd/root.go` (`const version`), used by both cobra and the update checker.
- `PersistentPreRun` reads the update cache (fast, local file) and spawns `update.RefreshCache()` as a goroutine (async, non-blocking). `PersistentPostRun` prints the update notice to stderr if one was set.
- Update cache stored in `~/.config/create-bug/update_check.json`; refreshed at most once per 24h. Version source: GitHub tags API.
- `--update` fetches latest version synchronously and runs `go install github.com/gabrielluong/create-bug@latest` if outdated. Note: this installs only the CLI; to also update the TUI run `go install github.com/gabrielluong/create-bug/...@latest`.

**Key conventions — TUI:**
- `internal/tui/form/model.go` owns all form state and submission. Focus zones: Summary → Component → Description → Blocks → Depends on → Whiteboard → Keywords → Submit.
- Up/Down arrows navigate between fields unless a dropdown is open or Description is focused (textarea captures arrows for line movement).
- `component.FuzzySelect`: single-value selector. Clears input on focus to show full list; auto-confirms best match on blur; restores previous selection if user clears input without choosing.
- `component.BugIDInput`: multi-value comma-separated input. Fuzzy-matches against current segment (text after last comma); excludes already-entered IDs; `[meta]` bugs ranked first then newest-first; selecting from dropdown appends `id, ` and re-opens for next entry.
- `component.MultiStringInput`: same pattern as `BugIDInput` but for string suggestions (used by Keywords). Sourced from `suggestions.LoadKeywords()`.
- Whiteboard uses `component.FuzzySelect` (single value); Keywords uses `component.MultiStringInput` (multi-value). Both source from `internal/suggestions`.
- Component selector is populated asynchronously: `Init()` dispatches `fetchComponentsCmd` which checks the cache first and falls back to the Bugzilla API via `EnsureComponents()`. A `componentsFetchedMsg` is received in `Update()` to swap in the populated selector. If fetch fails, the selector falls back to free-text input.
- `bugzilla.GetCachedComponents()` is the single source of truth for component lists at runtime — both CLI completion and TUI fuzzy selector read from it. Components are sorted alphabetically (as returned by the API).
- History autocomplete ranking (both CLI and TUI): bugs with `[meta]` in summary first, then `CreatedAt` descending.
- On successful submit, history is appended immediately so the new bug appears in Blocks/Depends on autocomplete when filing another.
- TUI uses hex colors (`#7C3AED` violet family, `#1E293B` slate surfaces) — keep the palette in `internal/tui/theme.go`.

## Claude Code Skill

`skills/create-bugs/SKILL.md` provides the `/create-bugs` slash command. It accepts three input modes:
- **File path** — reads a PLAN.md or similar markdown file
- **Inline text** — uses text passed directly as the argument
- **No argument** — derives work items from the current conversation (tasks discussed, decisions made, work completed)

Workflow:
1. Determine source and extract discrete work items
2. Run `create-bug --history --json` to identify already-filed bugs and skip them
3. Asks upfront for a tracking/metabug to `--blocks` against
4. Shows a dry-run breakdown (including skipped items) with dependency annotations; waits for confirmation
5. Files bugs sequentially via `create-bug --json`, using `--depends-on` only where tasks are truly blocked and `--blocks <metabug_id>` on every bug if provided
6. Reports each bug as `Bug <id> - <summary>` and prints a final summary

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
