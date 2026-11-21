---
name: create-bugs
description: Use this skill when the user asks to "file bugs", "create bugs", "create Bugzilla tickets", "file these as bugs", or wants to convert a plan, a discussion, or completed/in-progress work into Bugzilla bugs. Works from a PLAN.md file, inline markdown, or the current conversation context when no argument is given.
version: 1.1.0
argument-hint: <path/to/PLAN.md, inline markdown, or omit to derive from conversation>
allowed-tools: [Read, Bash(create-bug:*)]
---

Convert work — whether from a plan file, inline description, or the current conversation — into sequenced Bugzilla bugs.

## Step 1: Determine the Source

The user provided: $ARGUMENTS

- **File path** → read the file and use its contents as the work description.
- **Inline markdown / text** → use it directly.
- **Nothing provided** → derive the work items from the current conversation: review what has been discussed, designed, implemented, or decided, and identify discrete units of work that should be tracked as bugs.

## Step 2: Check Filing History

Run `create-bug --history --json` and compare the result against the work items you identified. Each entry has `id`, `summary`, `product`, `component`, `url`, and `created_at`. If any item appears to have already been filed (matching summary or clearly the same task), mark it as "already filed: Bug <id> - <summary>" and exclude it from the breakdown. If the history is empty or there are no matches, proceed.

Keep the history output around — it is also used to suggest a metabug (Step 4) and as a hint for the product/component the user usually files against (Step 3).

## Step 3: Extract Work Items

From the source (file, text, or conversation), extract:
1. Phases or logical groupings if present (h2/h3 headings, "Phase N" markers, natural topic boundaries in a conversation)
2. Individual tasks — each becomes one bug
3. Explicit dependencies: only add `depends-on` when a task truly cannot start until another is complete
4. Product and component for each bug:
   - If the user's config has defaults, omit `--product` / `--component` and let the defaults apply.
   - Otherwise infer from the source or from the product/component of recent history entries; ask if still unclear.
   - `--component` is fuzzy-matched against the product's live component list, so a close name (e.g. `toolbar`) is fine — but always confirm the resolved name in Step 5.
5. Optional fields, only when the source clearly calls for them:
   - `--whiteboard` (single value, e.g. `[good-first-bug]`) — any string is accepted.
   - `--keywords` (comma-separated, e.g. `regression,crash`). Don't add `meta` to work bugs.

**Scoping rule:** One bug = one discrete, reviewable unit of work. For conversation-derived bugs, focus on concrete actions taken or decisions made, not discussion or background context.

## Step 4: Ask About the Metabug

Before showing the breakdown, ask:

> "Is there a tracking or metabug these bugs should block? (e.g., `Bug 12000 - [meta] Implement feature X`)"

If the history from Step 2 contains bugs with `[meta]` in the summary, list the most recent ones as suggestions.

If the user provides one, every filed bug will include `--blocks <metabug_id>`.

## Step 5: Validate and Show Breakdown (Dry Run)

First validate every bug with `--dry-run` — this runs all validation and component resolution without filing anything or touching history:

```
create-bug "<summary>" [-p <product>] [-c <component>] --type <type> --description "<description>" [-w <whiteboard>] [-k <keywords>] --dry-run
```

It prints `{"dry_run": true, "params": {...}}`. Use `params.product` and `params.component` as the resolved values shown in the breakdown. If a dry run fails (e.g. no matching component, missing required option), fix the inputs or ask the user before continuing. If the resolved component differs from what you intended, call it out.

For dry runs, omit `--depends-on` values that refer to bugs not yet filed — those IDs are only known in Step 6.

Then present the full breakdown **before filing anything**. Include any already-filed bugs as skipped items so the user has full visibility:

```
## Bug Breakdown

Product :: Component: <Product> :: <Component>   ← or per-bug if they differ
Blocks: Bug 12000 - [meta] <feature name>

**Phase 1: <Name>**
⏭ Bug 12340 - <summary> (already filed — skipping)

1. [Bug] <Summary>
   - description: <one sentence>
   - depends-on: (none)
   - type: task|enhancement|defect

2. [Bug] <Summary>
   - description: <one sentence>
   - depends-on: Bug 1  ← only if Bug 2 truly can't start without Bug 1
   - type: task
   - keywords: regression  ← only if set

**Phase 2: <Name>**
3. [Bug] <Summary>
   - depends-on: (none — parallel with Bug 4)

Total: N bugs to file (M already filed)
```

Ask: "Does this breakdown look right? Any bugs to add, split, merge, or re-order?"

**Do not file any bugs until the user confirms.**

## Step 6: File Sequentially

For each bug in order (skipping already-filed ones):
1. Run:
   ```
   create-bug "<summary>" [-p <product>] [-c <component>] --type <type> --description "<description>" [-d <id,...>] [-b <metabug_id>] [-w <whiteboard>] [-k <keywords>] --json
   ```
2. Capture `{"id": N, "url": "...", "summary": "..."}` from stdout (stderr may contain a `Matched component:` notice — that's informational).
3. Map logical position → real bug ID (for subsequent `--depends-on` values)
4. Report inline: `Bug <id> - <summary> — <url>`

Quote arguments carefully: wrap summaries and descriptions in double quotes and escape any embedded `"`, `$`, or backticks.

On failure: stop, report the error, show which bugs were already filed, and ask how to proceed.

## Step 7: Summary Report

```
## Filed Bugs

**Phase 1: <Name>**
- Bug 12345 - <summary> — <url>
- Bug 12346 - <summary> — <url> (depends on Bug 12345)

Already filed (skipped):
- Bug 12340 - <summary>

All bugs block: Bug 12000 - [meta] <feature name>

Total: N bugs filed, M skipped.
```
