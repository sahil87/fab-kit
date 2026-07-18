# Intake: Help Examples — cobra `Example:` blocks on user-facing commands

**Change**: 260717-b91h-help-examples
**Created**: 2026-07-18

## Origin

One-shot `/fab-new b91h` from the backlog:

> - [ ] [b91h] 2026-07-18: Toolkit principle №3 (help is layered — short summary + examples) — no `fab` command populates cobra's `Example:` field (grep → zero hits); help is `Short` + occasional prose `Long` but carries no runnable example invocations. Deferred: add `Example:` blocks to the multi-flag user-facing commands (`batch archive`, `batch switch`, `config init`, `config show`, `resolve`, `score`, `dispatch start`). Spans ~7 commands; deferred per change 260717-ptwh (principles audit) as a coherent help-polish workstream rather than a single-command fix. shll v0.0.23.

Deferred item `[b91h]` from the toolkit-standards conformance audit (change `260717-ptwh`, conformance-report.md row P3: "layered `-h` has `Short` + occasional `Long` but **no command uses cobra `Example:`** — help lacks example invocations. Spans ~7 user-facing commands → deferred `[b91h]`").

The governing standard was re-fetched live at intake time (`shll standards principles`, installed shll **v0.1.0** — newer than the v0.0.23 the audit pinned; the №3 text is unchanged): *"Help text is layered — a short summary at the top, concrete usage examples after the flags — so a reader can drill from 'what is this' to 'how do I invoke it' without a manual."*

## Why

1. **Pain point**: `fab`'s help is `Short:` one-liners plus occasional `Long:` prose — no command shows a single runnable invocation. For the multi-flag commands (`config init` has 6 flags across two mutually-exclusive modes; `resolve` has 5 output-mode flags plus a socket flag; `dispatch start` reads its prompt from stdin, which no flag listing conveys), an agent or human reading `-h` learns *what the flags are* but not *how a real call is shaped*. Principle №3's layering obligation (summary → details → examples) is only half-met, and the ptwh audit graded it a GAP.
2. **Consequence of not fixing**: help stays a flag reference rather than a usage contract; the shll.ai-rendered command reference (built from `help-dump`, which byte-preserves each command's `-h` text) publishes the same example-free pages; callers keep inferring invocation shape from skill docs (`_cli-fab.md`) instead of from the tool itself — exactly the drift principle №3 exists to prevent.
3. **Why this approach**: populate cobra's native `Example:` field on the 7 audit-named commands. It renders in `-h` automatically, flows into `help-dump` output with zero extra plumbing (each node's `text` is the raw UsageString), and needs no custom help template. Deferred from ptwh as one coherent help-polish workstream rather than 7 drive-by edits — this change is that workstream.

## What Changes

Populate the `Example:` field on exactly these 7 cobra command definitions (no signature, flag, or behavior changes — help text only), plus one conformance test.

Cobra renders a populated `Example:` under an `Examples:` heading in `-h` output. Format conventions for all blocks: two-space indent per line (cobra convention), a `#` comment line above each invocation, 2–4 examples per command covering the primary flag combinations. The blocks below are the intended content — apply may tighten wording but should preserve the covered flag combinations.

### 1. `fab batch archive` — `src/go/fab/cmd/fab/batch_archive.go:40`

Flags: `--yes/-y`, `--dry-run`; args `[change...]`.

```go
Example: `  # Preview what would be archived, without archiving
  fab batch archive --dry-run

  # Archive all archivable changes without prompting
  fab batch archive --yes

  # Archive two specific changes (4-char ID or folder substring)
  fab batch archive b91h ptwh`
```

### 2. `fab batch switch` — `src/go/fab/cmd/fab/batch_switch.go:19`

Flags: `--list`, `--all`; args `[change...]`.

```go
Example: `  # Show available changes
  fab batch switch --list

  # Open tmux tabs in worktrees for two changes
  fab batch switch b91h ptwh

  # Open tabs for all changes
  fab batch switch --all`
```

### 3. `fab config init` — `src/go/fab/cmd/fab/config.go:477`

Flags: `--system`, `--project`, `--name`, `--description`, `--source-path` (repeatable), `--test-path` (repeatable). Two mutually-exclusive modes — examples must show both.

```go
Example: `  # Write the ~/.fab-kit/config.yaml system-layer scaffold
  fab config init --system

  # Generate fab/project/config.yaml from the field registry
  fab config init --project --name my-app --description "My application"

  # Seed source/test paths (repeatable flags)
  fab config init --project --name my-app --source-path src/ --test-path '**/*_test.go'`
```

### 4. `fab config show` — `src/go/fab/cmd/fab/config.go:146`

Flags: `--origin`.

```go
Example: `  # Print the effective (post-cascade) config
  fab config show

  # Annotate each field with its provenance (project / system / default)
  fab config show --origin`
```

### 5. `fab resolve` — `src/go/fab/cmd/fab/resolve.go:14`

Flags: `--id` (default), `--folder`, `--dir`, `--status`, `--pane`, `--server/-L`; arg `[change]` (defaults to the active change).

```go
Example: `  # 4-char ID of the active change (default output)
  fab resolve

  # Full folder name for a change reference
  fab resolve --folder b91h

  # Path to the change's .status.yaml
  fab resolve --status b91h

  # tmux pane ID, targeting a specific tmux socket
  fab resolve --pane -L work b91h`
```

### 6. `fab score` — `src/go/fab/cmd/fab/score.go:15`

Flags: `--check-gate`, `--stage` (default `intake`); arg `<change>`.

```go
Example: `  # Compute and persist the intake confidence score
  fab score b91h

  # Read-only gate check — exits non-zero below the threshold
  fab score --check-gate --stage intake b91h`
```

### 7. `fab dispatch start` — `src/go/fab/cmd/fab/dispatch_start.go:20`

Flags: `--timeout`; args `<change> <stage>`; prompt read from **stdin** — the examples are the only place `-h` can convey that shape.

```go
Example: `  # Launch the apply stage's dispatch command, prompt on stdin
  fab dispatch start b91h apply < prompt.md

  # Enforce a 30-minute POSIX timeout inside the launch wrapper
  fab dispatch start --timeout 1800 b91h apply < prompt.md`
```

### 8. Conformance test

New test (e.g., `src/go/fab/cmd/fab/examples_test.go`, following the existing per-surface test-file naming) walking the real command tree and asserting each of the 7 target commands has a non-empty `Example`. This pins the contract surface the same way the help-dump standard's "keep a minimal test pinning the above" guidance does, and satisfies the project rule that Go changes ship tests. Formatting assertions (every non-blank line indented two spaces) MAY be included but are secondary.

### Explicit non-changes (dispositions to preempt strict review)

- **No `_cli-fab.md` update**: the constitution requires `_cli-fab.md` updates for "new or changed command signatures" — this change alters zero signatures, flags, or behaviors; help text only. (`_cli-fab.md` documents invocation grammar, which is untouched.)
- **No custom help template**: `Examples:` renders where cobra's stock template puts it — after Usage/Aliases, *before* the Flags section — not literally "after the flags" as the principle's phrasing suggests. Verified at intake time that **no toolkit tool populates examples today** (walked `help-dump` trees of shll v0.1.0, wt, idea, run-kit — zero `Examples:` sections), so there is no reference placement to match; principle №7 (compose, don't reinvent) favors the native field over a bespoke template, and the enforcement receipt for №3 is help-dump conformance, not example placement. If the toolkit later standardizes placement, that lands in shll's standards repo first.
- **No manual docs/site or shll.ai work**: `help-dump` byte-preserves each command's `-h` text into the published reference, so the examples propagate on the next release automatically (help text is a release artifact). The fab `help-dump` tests use synthetic command trees and are unaffected.
- **No sweep beyond the 7 commands**: the audit deliberately scoped to "multi-flag user-facing commands"; single-flag and internal/hidden commands stay example-free in this change.

## Affected Memory

- `distribution/distribution`: (modify) § Toolkit Standards Conformance — update the principle №3 posture: the `[b91h]` deferred gap is closed (Example: blocks shipped on the 7 audit-named commands); layered-help obligation now met. Frontmatter `description` and the domain index row updated to match (regenerate via `fab memory-index`).

## Impact

- **Code**: 6 files under `src/go/fab/cmd/fab/` (`batch_archive.go`, `batch_switch.go`, `config.go` — two command sites, `resolve.go`, `score.go`, `dispatch_start.go`) — each gains one `Example:` field; zero behavior change.
- **Tests**: 1 new test file (`examples_test.go` or similar) pinning non-empty `Example` on the 7 commands. Existing tests unaffected (helpdump tests use synthetic trees; no test pins real `-h` bytes in this repo).
- **Docs/memory**: `docs/memory/distribution/distribution.md` posture update + index regen at hydrate.
- **Downstream**: shll.ai command reference picks the examples up on the next release via its scheduled `help-dump` pull — no push-side work.
- **Backlog**: `[b91h]` marked done at archive time (handled by `/fab-archive`).

## Open Questions

*(none — the backlog entry + ptwh audit report fully scope the change; all residual decisions graded Confident or better below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly the 7 audit-named commands (`batch archive`, `batch switch`, `config init`, `config show`, `resolve`, `score`, `dispatch start`) | Backlog entry and ptwh conformance report both enumerate them; widening later is trivial | S:85 R:80 A:85 D:80 |
| 2 | Confident | Use cobra's native `Example:` field with stock template placement (renders before Flags), not a custom help template to literally satisfy the principle's "after the flags" phrasing | Backlog says "add `Example:` blocks"; verified live (shll v0.1.0 + wt/idea/run-kit help-dumps) that no toolkit tool populates examples, so no placement precedent exists; principle №7 favors the native field; №3's enforcement receipt is help-dump conformance, not placement | S:70 R:85 A:60 D:65 |
| 3 | Confident | Example block format: two-space indent, `#` comment line above each invocation, 2–4 examples per command covering primary flag combos | Cobra's two-space indent is the ecosystem convention; style otherwise unspecified and trivially editable later | S:55 R:90 A:70 D:60 |
| 4 | Confident | Ship a conformance test asserting non-empty `Example` on the 7 commands; make no `_cli-fab.md` change (zero signature changes) | "Go changes ship tests" is a project review rule; `_cli-fab.md` constraint keys on command *signatures*, which are untouched — disposition stated in-intake to preempt strict review | S:60 R:85 A:80 D:70 |
| 5 | Confident | Memory update scoped to `distribution/distribution.md` § Toolkit Standards Conformance (posture: №3 `[b91h]` gap closed) + index regen | That section owns the audit posture and already tracks the four deferred backlog items; no other domain documents help output | S:65 R:85 A:80 D:75 |

5 assumptions (1 certain, 4 confident, 0 tentative, 0 unresolved).
