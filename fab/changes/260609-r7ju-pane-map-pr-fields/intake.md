# Intake: Add PR URL and Number to `fab pane map` JSON Output

**Change**: 260609-r7ju-pane-map-pr-fields
**Created**: 2026-06-09
**Status**: Draft

## Origin

Initiated one-shot from a fully-specified prompt (Context / Requirements / Tests /
Affected sections supplied verbatim by the user). The request is driven by an
external consumer — **run-kit's sidebar** wants to surface the active change's PR
without re-deriving it.

> Add PR URL and number to `fab pane map` JSON output.
>
> run-kit's sidebar wants to surface the active change's PR. The PR URL already
> lives in each change's `.status.yaml` under the `prs:` list — but `fab pane map`
> currently drops it. `panemap.go` already loads the status file (the
> `sf.Load(statusPath)` call in `resolvePane`, used to derive `stage`), so the PR
> URL is one field away at a seam we already pay for. No new file reads, no network.

The prompt's codebase claims were verified before intake generation:
- `resolvePane` (`panemap.go:321`) already calls `sf.Load(statusPath)` and derives
  `stage` via `status.DisplayStage` — the status file is already in hand at this seam.
- The statusfile package **already exposes** the PR list as a public field:
  `StatusFile.PRs []string` (`statusfile.go:105`, decoded at `:162`, encoded at `:387`).
  → No new accessor is needed. The intake's "if not already present, add a minimal
  accessor" hedge resolves to "already present" — use `statusFile.PRs` directly.
- `paneJSON` already uses the `*string` + `toNullable` nil convention (`panemap.go:347-366`)
  for `Repo`/`Change`/`Stage`, which the new fields mirror.
- Canonical PR URL shape in the repo is `https://github.com/org/repo/pull/<n>`
  (`docs/specs/skills.md:746`), matching the `/pull/<n>` parse target.

## Why

1. **Problem**: run-kit's sidebar needs the active change's PR URL/number to render a
   link. The data already exists on disk in `.status.yaml`'s `prs:` list, but
   `fab pane map --json` drops it, forcing run-kit to either re-read every change's
   `.status.yaml` itself or do without.
2. **Consequence if unfixed**: run-kit duplicates a `.status.yaml` read that fab-kit
   already performs per pane, or the sidebar can't show the PR at all. fab-kit is the
   natural owner of "what's on disk for this pane" — leaving the field out makes the
   JSON contract incomplete for its one machine consumer.
3. **Why this approach**: The status file is **already loaded** in `resolvePane` for
   stage derivation. Surfacing `prs[-1]` and its parsed number is a near-zero-cost
   addition at a seam we already pay for — no new file read, no network call. The
   alternative (run-kit reading `.status.yaml` itself) duplicates I/O and couples
   run-kit to fab's on-disk schema. Surfacing through the existing JSON contract keeps
   the schema dependency on fab's side, where it belongs.

   **Deliberate boundary**: PR *status* (open/merged/closed, CI state) is **not**
   fab-kit's job and is explicitly out of scope. fab-kit surfaces only the URL/number
   already written to disk by `/git-pr`; run-kit fetches live status separately and
   caches it. `fab pane map` stays filesystem-only and poll-free — no `gh`, no `git`,
   no network.

## What Changes

### 1. `paneRow` gains two fields (`panemap.go`)

`paneRow` (the internal resolved-row struct, `panemap.go:44-54`) gains a single
**string** field; the PR number is parsed at the JSON boundary (see §5), keeping
`paneRow` string-only and consistent with its siblings:

```go
type paneRow struct {
    // ...existing fields...
    prURL string // last entry in .status.yaml prs:, "" when absent/empty/unresolved
}
```

<!-- clarified: pr_number null representation — user chose parse-at-JSON-boundary over a *int on paneRow. paneRow carries only prURL string; parsePRNumber runs in printPaneJSON producing *int (nil = null). Identical JSON either way; keeps paneRow string-only. -->

> **Null semantics**: `paneRow` carries only `prURL string`; the `toNullable`
> convention converts `""` → JSON null at `printPaneJSON`. The PR *number* is parsed
> there too (§5), so a nil `*int` (= null) cleanly represents both "no URL" and
> "unparseable URL" without a sentinel int.

### 2. `paneJSON` gains two JSON fields (`panemap.go:347-358`)

```go
type paneJSON struct {
    // ...existing fields...
    PRURL    *string `json:"pr_url"`
    PRNumber *int    `json:"pr_number"`
}
```

These are appended after the existing fields. Field/JSON-tag naming follows the
existing snake_case convention (`window_index`, `agent_state`).

### 3. `resolvePane` sources the URL from the already-loaded status file (`panemap.go:317-326`)

Inside the existing `if statusFile, err := sf.Load(statusPath); err == nil { ... }`
block — the SAME load already used for `stage` — read `statusFile.PRs`:

```go
if statusFile, err := sf.Load(statusPath); err == nil {
    stage, _ := status.DisplayStage(statusFile)
    stageName = stage
    if n := len(statusFile.PRs); n > 0 {
        prURL = statusFile.PRs[n-1]        // last = most recent
    }
}
```

No second `.status.yaml` read. Panes with no fab change, no fab dir, unresolved git,
or an empty/absent `prs:` list leave `prURL` empty (→ both JSON fields null). The PR
number is NOT parsed here — it's derived at the JSON boundary (§5).

### 4. PR-number parse helper (`panemap.go`)

A small pure function parses the trailing `/pull/<n>` segment:

```go
// parsePRNumber extracts the PR number from a GitHub PR URL's trailing
// /pull/<n> segment. Returns (n, true) on success, (0, false) when the URL
// has no parseable /pull/<n> segment.
func parsePRNumber(url string) (int, bool) {
    // find "/pull/" then strconv.Atoi the segment up to the next "/" or end
}
```

- Input `https://github.com/org/repo/pull/42` → `(42, true)`
- Input `https://github.com/org/repo/pull/42/files` → `(42, true)` (trailing path tolerated)
- Input with no `/pull/<n>` (malformed) → `(0, false)` → `pr_number: null`, `pr_url` still set
- Trailing non-numeric (`/pull/abc`) → `(0, false)`

### 5. `printPaneJSON` parses + maps to nullable JSON (`panemap.go:384-404`)

This is where the PR number is derived. Map `prURL` onto `PRURL *string` via
`toNullable`; run `parsePRNumber(r.prURL)` and set `PRNumber *int` to `&n` on success,
`nil` otherwise:

```go
prURL := toNullable(r.prURL)
var prNum *int
if n, ok := parsePRNumber(r.prURL); ok {
    prNum = &n
}
out[i] = paneJSON{ /* ...existing... */, PRURL: prURL, PRNumber: prNum }
```

`pr_url` is null when the URL is empty; `pr_number` is null when there's no URL OR the
URL is unparseable. (Parsing `""` returns `(0, false)`, so an empty URL naturally
yields both-null without a special case.)

### 6. Table output is UNCHANGED

`printPaneTable` (`panemap.go:407-462`) is **not touched**. No new columns. The plain
table is the human glance; JSON is the machine contract. A test asserts byte-identity.

### 7. Docs: `fab pane map` JSON shape

Update the `_cli-fab.md` reference (and any spec doc describing the `fab pane map` JSON
shape) to document the two new fields, their types (`pr_url: string|null`,
`pr_number: number|null`), null semantics, and the explicit non-goal (no PR status, no
network). Per the constitution, CLI signature/output changes update `_cli-fab.md`.

## Affected Memory

- `runtime/pane-commands`: (modify) `fab pane map` JSON output gains `pr_url` and
  `pr_number` fields sourced from `.status.yaml` `prs:` — document the fields, null
  semantics, and the filesystem-only / no-network boundary. (Confirmed against
  `docs/memory/runtime/index.md`: `pane-commands.md` owns the `fab pane {map,...}`
  subcommand reference and JSON shape.)

## Impact

- **`src/go/fab/cmd/fab/panemap.go`** — `paneRow` (+2 fields), `paneJSON` (+2 fields),
  `resolvePane` (read `statusFile.PRs` in the existing load block), `printPaneJSON`
  (map new fields), new `parsePRNumber` helper.
- **`src/go/fab/cmd/fab/panemap_test.go`** — new unit tests (see Tests below).
- **`src/go/fab/internal/statusfile/`** — **no change expected**; `StatusFile.PRs` is
  already public. (If the plan finds a reason to add a convenience accessor, it would
  go here, but the field is directly usable.)
- **`src/go/fab/internal/status/`** — no change expected; `DisplayStage` is the pattern
  to mirror for read-only derivation, but `PRs` needs no new derivation function.
- **`_cli-fab.md`** (kit source: `src/kit/skills/_cli-fab.md`) + any `fab pane map` JSON
  shape doc — document the new fields.
- **Consumer**: run-kit (external) — gains the fields; not modified in this repo.
- **No network, no `gh`/`git` subprocess, no second file read** — invariants preserved.

## Open Questions

- **Q1 (`pr_number` null representation) — RESOLVED**: User chose to parse the number at
  the JSON boundary (option b). `paneRow` carries only `prURL string`; `parsePRNumber`
  runs in `printPaneJSON` producing a `*int` (nil = null). Identical JSON to the
  alternative; keeps `paneRow` string-only and confines parsing to the JSON path.

_No open questions remain._

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Source `pr_url`/`pr_number` from the `statusFile` already loaded in `resolvePane`; no second `.status.yaml` read. | Explicit hard requirement in the prompt; verified `sf.Load` at `panemap.go:321` is the seam. | S:98 R:90 A:95 D:98 |
| 2 | Certain | Use the existing `StatusFile.PRs []string` field directly — no new statusfile accessor. | Verified `PRs` is already public (`statusfile.go:105`); the prompt's "add accessor if absent" hedge resolves to absent-of-need. | S:95 R:85 A:98 D:95 |
| 3 | Certain | `pr_url` = LAST entry of `prs:` (most recent); null when list absent/empty. | Explicit requirement; matches `AddPR` append-order semantics (`status.go:385`). | S:98 R:88 A:95 D:98 |
| 4 | Certain | Reuse the `toNullable` / `*string` / `*int` nil convention for JSON null. | Explicit requirement; `paneJSON` already uses it (`panemap.go:353-358`). | S:98 R:90 A:98 D:98 |
| 5 | Certain | Table output unchanged — JSON-only fields, no new columns; assert byte-identity in a test. | Explicit requirement; `printPaneTable` is independent of `paneJSON`. | S:98 R:92 A:95 D:98 |
| 6 | Certain | No `gh`/`git`/network; `fab pane map` stays filesystem-only and poll-free. | Explicit, emphatic requirement; current code reads only the local status file. | S:99 R:85 A:98 D:99 |
| 7 | Confident | `pr_number` parsed from the trailing `/pull/<n>` segment; null when no URL or unparseable; `pr_url` still set on a malformed URL. | Explicit requirement; PR URL shape `…/pull/<n>` is the repo convention (`skills.md:746`). Edge handling (trailing path, non-numeric) is the obvious robust default. | S:90 R:80 A:88 D:85 |
| 8 | Confident | Document the new fields + non-goal in `_cli-fab.md` (kit source) and the JSON-shape doc. | Constitution requires CLI output changes update `_cli-fab.md`; the exact JSON-shape doc location confirmed at hydrate. | S:85 R:85 A:85 D:80 |
| 9 | Certain | Represent `pr_number` nullability by parsing in `printPaneJSON` rather than a `*int` on `paneRow` (Q1, option b). | Clarified — user chose parse-at-JSON-boundary. Both options yield identical JSON; `paneRow` stays string-only. | S:95 R:88 A:80 D:95 |
| 10 | Certain | Affected memory file is `runtime/pane-commands` (owns the `fab pane {map,...}` JSON-shape reference). | Confirmed against `docs/memory/runtime/index.md` during intake — `pane-commands.md` documents the `fab pane map` subcommand and its JSON output. | S:90 R:85 A:95 D:90 |

10 assumptions (8 certain, 2 confident, 0 tentative, 0 unresolved).
