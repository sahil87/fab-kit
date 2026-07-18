# Intake: Usage-Error Exit Codes (Toolkit Principle №4)

**Change**: 260717-swon-usage-error-exit-codes
**Created**: 2026-07-18

## Origin

> /fab-new swon

One-shot invocation from the backlog. `fab/backlog.md` entry `[swon]` (2026-07-18), verbatim:

> Toolkit principle №4 (fail fast) — exit-code convention. `main.go:51-56` maps EVERY error to `os.Exit(1)` (`SilenceUsage/SilenceErrors:true`, no `SetFlagErrorFunc`), so cobra usage/argument errors (unknown flag, arg-count violation, mutually-exclusive conflict — verified: `fab score` no-arg, unknown flag, and a nonexistent-change resolve ALL exit 1) are indistinguishable from operational failures. The toolkit convention is `0` success / `1` operational failure / `2` usage error. Deferred (restructuring): route cobra usage errors to exit 2 via `SetFlagErrorFunc` / a usage-error sentinel in `main.go`, reconciled with the EXISTING domain-specific exit-2 uses that would otherwise collide — `memory_index` (exit 2 = destructive loss, `memory_index.go:264`) and the pane family (`paneValidationExitCode`: 2 = pane-missing, 3 = tmux-failure, pinned by `pane_exitcode_test.go`). Error *wording* already conforms (what-failed/why/what-next). Deferred per change 260717-ptwh (principles audit) — restructuring gap, not small-additive. shll v0.0.23.

Deferred from change 260717-ptwh (toolkit principles audit, shipped as PR #486 "fix: Toolkit Standards Conformance") because it is a restructuring gap, not a small-additive fix. No prior conversation context beyond the backlog entry.

## Why

1. **The pain point**: the `fab` binary's `main()` (`src/go/fab/cmd/fab/main.go:51-56`) maps every error returned by `Execute()` to `os.Exit(1)`. The root command sets `SilenceUsage: true` / `SilenceErrors: true` and installs no `SetFlagErrorFunc`, so cobra usage errors — unknown flag, arg-count violation (`fab score` with no args), mutually-exclusive flags-group conflict (`fab resolve --status --folder`), unknown subcommand — are indistinguishable from operational failures (missing change, failed preflight, tmux errors). An agent or script cannot branch on failure class.

2. **The consequence**: this violates toolkit principle №4 (fail fast with actionable errors, obligation MUST): "Exit codes are documented per subcommand and mean something — `0` success, `1` operational failure, `2` usage error is the toolkit convention." The constitution's Toolkit Standards article (v1.4.0, added by 260717-y8it) binds this repo to that standard. Per the standard's failure mode: "an undocumented exit code turns every script's error handling into guesswork."

3. **Why this approach**: the standard's own enforcement receipt names the mechanism — "the shared `errSilent`/`errExitCode` sentinel pattern (exit codes without double-printed messages)". A usage-error sentinel classified in `main()` mirrors the pattern the toolkit already uses (and the classification-by-error-value discipline fab already follows in `paneValidationExitCode`: "Classification rides on the error value — no string matching"). Error *wording* already conforms (what-failed/why/what-next, verified in 260717-ptwh) — this change is exit-code mapping only.

## What Changes

### 1. Usage-error classification in the `fab` binary's `main()`

Current code (`src/go/fab/cmd/fab/main.go:51-56`):

```go
func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
```

New behavior: introduce a usage-error sentinel (e.g. an unexported `usageError` wrapper type in `package main`); `main()` unwraps via `errors.As` and exits **2** for usage errors, **1** for everything else. The `ERROR: %s` stderr print is unchanged (wording already conforms; the sentinel must not double-print — that is the point of the toolkit's sentinel pattern).

### 2. Usage-error classes routed to exit 2

All of these exit 1 today (first three verified in the 260717-ptwh audit); all MUST exit 2:

| Class | Example | Candidate seam |
|-------|---------|----------------|
| Unknown/malformed flag | `fab score --nope` | `root.SetFlagErrorFunc` wrapping the error in the sentinel |
| Arg-count violation | `fab score` (no args; cobra `Args` validators like `ExactArgs`) | wrap each command's `Args` validator via a tree walk after assembly in `newRootCmd()` |
| Unknown subcommand | `fab nonsense` | root's `legacyArgs` error — caught by the same `Args` wrapping applied at the root |
| Mutually-exclusive flags-group conflict | `fab resolve --status --folder` (`MarkFlagsMutuallyExclusive`) | cobra's `ValidateFlagGroups` returns a plain error mid-`execute()` with no public hook — exact seam decided at apply |

An alternative **unified** mechanism is also on the table for apply to evaluate: a "did-any-RunE-start" sentinel (a root `PersistentPreRun` sets a flag; any `Execute()` error surfacing before any command's run phase began is classified as usage). Either mechanism satisfies the requirement; the requirement is the *behavior* (the four classes above exit 2), not the seam. Whatever the seam, classification MUST ride on error values or execution phase — no stderr/message string matching.

**Stays exit 1 (operational, NOT usage)**: nonexistent-change resolution (`fab resolve nope` — valid syntax, missing data; the backlog lists it only to demonstrate today's indistinguishability), preflight validation failures, `fab score --check-gate` below-gate exits, tmux/gh/filesystem failures. Success remains 0. `fab log command`'s always-exits-0 contract is unaffected (its cobra arg-count errors move 1→2, still "non-zero before RunE").

### 3. Reconciliation with existing domain-specific non-1 exit codes (the collision)

Both existing schemes are **unchanged** — no renumbering:

- **Pane family** (`capture`, `send`, `window-name`): `2` = pane missing, `3` = any other tmux failure (`paneValidationExitCode` in `pane.go:14`, `tmuxExitCode` in `pane_window_name.go:132`; pinned by `pane_exitcode_test.go`; documented in `_cli-fab.md` and `docs/memory/runtime/pane-commands.md`).
- **`fab memory-index --check`**: tiered `0`/`1`/`2`, where `2` = destructive loss (`memory_index.go:264`; consumed by the hydrate skills' refuse-before-regen guard, `docs/memory/memory-docs/hydrate.md`).

**Coexistence rule**: usage errors exit 2 binary-wide at parse/validation time (before the subcommand's handler runs); the domain schemes apply in-handler. For these subcommands exit 2 is therefore ambiguous between "usage error" and the domain meaning — accepted, because (a) principle №4's own instruction is "exit codes are documented per subcommand" (per-subcommand tables are the sanctioned shape, not a single global map), (b) a usage error is a static caller bug fixable at authoring time, not a runtime condition scripts branch on, and (c) stderr wording disambiguates. Renumbering was rejected: it breaks the pinned pane test contract and downstream consumers (operator/run-kit branching on pane codes; the hydrate guard branching on memory-index tier 2).

### 4. Documentation sweep (constitution: CLI change ⇒ `_cli-fab.md` + tests)

Grep-verified claims to update, plus the per-subcommand documentation the principle requires:

- `src/kit/skills/_cli-fab.md` — add the binary-wide exit-code convention (`0`/`1`/`2` + sentinel + coexistence rule) near the top; update stale claims: line ~224 ("cobra arg-count errors exit non-zero before RunE" → exit 2), line ~247 (`fab resolve` flags-group conflict "exits non-zero" → exit 2), line ~337 (`fab resolve-agent` usage error → exit 2); annotate the pane and memory-index exit-code sections with the usage-error coexistence note.
- `src/kit/skills/_preamble.md` — the two "cobra arg-count errors exit non-zero before RunE" claims (§ Common fab Commands table row + Key behaviors bullet, and the change-context `fab log command` step).
- SPEC mirrors under `docs/specs/skills/` per code-quality § Sibling & Mirror Sweeps — on a CLI change, treat **all** of a touched skill's SPEC mirrors as the sweep class; grep the old claims repo-wide (including user-facing string literals, per the recurring-lessons discipline) before finishing apply.
- `docs/site/` needs nothing: fab-kit has no CLI-surface exit-code page (`install.md`/`workflows.md` only, verified).

### 5. Tests (constitution: Go changes ship tests)

- Pin the new mapping with a table test mirroring `pane_exitcode_test.go`'s shape — requires a testable seam (e.g. extract main's body into a helper returning the exit code, or test the classifier directly): unknown flag → 2, no-arg `score` → 2, unknown subcommand → 2, flags-group conflict → 2, operational error (nonexistent change) → 1, success → 0.
- `pane_exitcode_test.go` must stay green unmodified (codes unchanged) — it is the regression guard for the no-renumbering decision.

## Affected Memory

- `distribution/kit-architecture`: (modify) record the fab binary's exit-code convention — `0` success / `1` operational / `2` usage, the sentinel classification in `main()`, and the coexistence rule with the in-handler domain schemes
- `runtime/pane-commands`: (modify) add the usage-error coexistence note to the pane-family exit-code scheme (exit 2 at parse time = usage error; in-handler 2/3 scheme unchanged)

## Impact

- **Code**: `src/go/fab/cmd/fab/main.go` (sentinel + exit mapping), likely `newRootCmd()` wiring for the chosen seams, one new test file. No pane/memory-index handler changes.
- **Kit docs**: `src/kit/skills/_cli-fab.md`, `src/kit/skills/_preamble.md`; SPEC mirrors under `docs/specs/skills/`.
- **Memory**: the two files above.
- **Consumers**: skills' generic failure rule ("non-zero exit → STOP") is unaffected — 2 is still non-zero. No behavior change for valid invocations. No user-data restructuring → no migration.
- **Out of scope**: the `fab-kit` and `fab-kit`-shim binaries (`src/go/fab-kit/` — their exit-1 mapping and `doctor`'s exit-with-failure-count aggregation), renumbering any existing scheme, error-message wording changes.

## Open Questions

- None — the backlog entry plus toolkit principle №4 resolve the direction; residual mechanism choices are graded below and decided at apply.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Coexistence over renumbering: pane (2/3) and memory-index (0/1/2) schemes stay unchanged; usage errors exit 2 binary-wide with the ambiguity documented per subcommand | Principle №4 sanctions per-subcommand exit-code tables; renumbering breaks the pinned pane test and downstream consumers (operator/run-kit, hydrate guard); usage errors are static caller bugs, not runtime branch conditions | S:55 R:70 A:70 D:65 |
| 2 | Confident | Mechanism: unexported usage-error sentinel unwrapped in `main()` via `errors.As` (no string matching), fed by per-class cobra seams (`SetFlagErrorFunc` + `Args`-validator tree-walk wrap); flags-group seam (no public cobra hook) and the alternative did-any-RunE-start unified mechanism decided at apply | Backlog names `SetFlagErrorFunc`/sentinel; the toolkit standard's receipt names the sentinel pattern; seam choice is internal and easily changed — apply decides-and-records | S:60 R:85 A:75 D:55 |
| 3 | Confident | Scope: `fab` binary only (`src/go/fab/`); `fab-kit` + shim binaries and `fab-kit doctor` aggregation out of scope | Backlog cites fab's `main.go:51-56` specifically; extending to the installer binaries later is additive, not blocked by this change | S:70 R:75 A:80 D:70 |
| 4 | Certain | Error wording unchanged — exit-code mapping only; `ERROR: %s` stderr print stays | Backlog states wording already conforms (verified in the 260717-ptwh audit); changing it would widen scope for no conformance gain | S:85 R:90 A:95 D:90 |
| 5 | Confident | Data-condition failures stay exit 1: nonexistent-change resolve, preflight failures, below-gate `--check-gate` exits are operational, not usage | The convention defines 2 as *usage* error (malformed invocation); a syntactically valid query for missing data is an operational failure — the backlog lists nonexistent-resolve only to show today's indistinguishability | S:60 R:80 A:85 D:75 |
| 6 | Confident | Per-subcommand exit-code documentation lands in `_cli-fab.md` (+ SPEC mirrors + the two memory files); no docs/site CLI-surface page is created | fab-kit's `docs/site/` has no CLI-surface page today (verified: `install.md`/`workflows.md` only); `_cli-fab.md` is the constitution-mandated CLI reference and already carries the pane/memory-index exit-code tables | S:65 R:85 A:80 D:70 |

6 assumptions (1 certain, 5 confident, 0 tentative, 0 unresolved).
