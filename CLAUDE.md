# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
go build -o create-bug .   # compile → ./create-bug
go run .                   # run without building
./create-bug               # run compiled binary
```

No test runner is configured yet.

## Architecture

Go CLI tool using spf13/cobra for creating Bugzilla bugs via the REST API. Designed for standalone use and as a Claude Code SKILL.

**Three-layer separation (for future Bubbletea TUI integration):**

```
main.go              → calls cmd.Execute()
cmd/
  root.go            → root cobra command (is the create-bug command itself)
  create_bug.go      → flag parsing, default merging, fuzzy resolution, output
internal/bugzilla/
  create_bug.go      → CreateBug() business logic entry point
  components.go      → hardcoded component lists (Fenix) + fuzzy resolution
internal/client/
  client.go          → HTTP transport: Client, CreateBugParams, error codes
internal/config/
  config.go          → Config, BugDefaults, Load() (env > file > defaults)
```

**Key conventions:**
- `internal/client/` owns all HTTP logic. New API calls go here.
- The root command *is* the create-bug command — no subcommands.
- `--json` flag for machine-readable stdout output (Claude SKILL integration).
- Errors to **stderr** via `fmt.Fprintf(os.Stderr, ...)`. `RunE` returns the error to trigger `os.Exit(1)`.
- `SilenceErrors` and `SilenceUsage` on root — Cobra won't double-print.
- Bugzilla returns `{ "error": true, "code": N, "message": "..." }` — check this **before** HTTP status.
- Requires `config.APIKey`; exit early if empty before any network call.
- Use `cmd.Flags().Changed("flag")` to distinguish "not passed" from "passed empty" when merging with config defaults.
- `--component` supports fuzzy matching via `sahilm/fuzzy` for known products (Fenix). Unknown products pass through for API validation.
- Tab completion registered for `--component` flag (returns known components when `--product` is set).

## Auth

Set `BUGZILLA_API_KEY` (and optionally `BUGZILLA_URL`) as environment variables, or write them to `~/.config/create-bug/config.json`:

```json
{
  "apiKey": "your-key",
  "baseUrl": "https://your-bugzilla-instance.example.com",
  "defaults": {
    "product": "Fenix",
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
