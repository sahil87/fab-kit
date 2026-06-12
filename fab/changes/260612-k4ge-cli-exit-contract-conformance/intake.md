# Intake: CLI Exit-Code Contract Conformance

**Change**: 260612-k4ge-cli-exit-contract-conformance
**Created**: 2026-06-12

## Origin

> k4ge

One-shot `/fab-new k4ge` (backlog ID). The entry lives at `fab/backlog.md:11` in the **main repo** (`/home/sahil/code/sahil87/fab-kit`) — this worktree's branch predates the entry, so its local backlog copy does not contain it. Source backlog entry (2026-06-12):

> Skills-audit batch 1/5 — CLI exit-code contract conformance (Go + tests + doc rows). SHIP FIRST — one item bypasses the pipeline's only safety gate. GOAL: make the binary honor the contracts the skills document, then fix the doc rows that were wrong about the binary.

Evidence source: `docs/specs/findings/skills-review-2026-06-12.md` §2 Theme 1 (+ Theme 4 for the stage-metrics item), line numbers vs commit `1431a9c3`. **That report exists only uncommitted in the main repo** — all needed findings are reproduced verbatim below so this intake is self-sufficient.

## Why

1. **The pipeline's only safety gate is bypassable.** `fab score --check-gate` is documented (in three places) to exit non-zero on gate fail, but the binary always exits 0 — so `/fab-ff` and `/fab-fff` cannot detect a failed intake gate via the documented contract. Empirically reproduced: `gate: fail` on stdout, exit 0. The single human-judgment checkpoint in the six-stage pipeline silently passes everything.
2. **The state machine can permanently brick a change.** `fab status advance ship|review-pr` writes `ready` and `skip intake` writes `skipped` — states the schema forbids for those stages — after which `fab preflight` exits 1 forever ("State 'ready' not allowed for stage ship"). Empirically reproduced.
3. **The generic failure rule amplifies every wrong doc row.** `_preamble.md`'s rule keys STOP on non-zero exit. A false "exits non-zero" claim means real failures silently pass; a false "exits 0" claim means benign paths abort. Agents copying the invalid canonical form `fab change resolve --folder` hit `unknown flag`, exit 1, and STOP.

If unfixed: ungated autonomous pipelines, brickable changes, and agents stopping (or not stopping) at the wrong moments. This batch ships first of five (w7dp explicitly depends on the fixed exit-code contract).

## What Changes

### 1. `fab score --check-gate` exits non-zero on gate fail (SHIP-FIRST item)

`src/go/fab/cmd/score.go` (reported as `cmd/fab/score.go:25-32`): the command prints the gate YAML and returns `nil` unconditionally. Change: when the gate result is `fail`, return a non-zero exit (keep the YAML on stdout). This makes the binary conform to `_preamble.md:230`, `_cli-fab.md:91`, and `_pipeline.md` Pre-flight 3 — those doc rows become true and stay untouched.

### 2. `lookupTransition` rejects schema-forbidden target states

`src/go/fab/internal/status/status.go:21-61`: transitions must validate the target state against `AllowedStates` for the target stage. Known violations to close (and cover with tests):

- `fab status advance ship` / `advance review-pr` write `ready` — forbidden for those stages
- `fab status skip intake` writes `skipped` — forbidden for intake

Result: the command errors cleanly instead of writing a state that bricks `fab preflight` permanently.

### 3. `fab change switch` Next: routing off-by-one

`src/go/fab/internal/change/change.go:217-223`: at post-review stages, switch prints `/fab-archive` where `/git-pr-review` is correct, diverging from `fab status` for the same state. Align switch's Next: derivation with fab-status (cf. `fab-switch.md:84-99`).

### 4. Archive exit semantics

- `fab change archive` with no argument currently exits 0 printing help text → enforce `cobra.ExactArgs(1)` (usage error, non-zero).
- **Re-archive soft skip is documented but unreachable**: `_cli-fab.md:42-46` and `fab-archive.md:110` document a soft skip, but genuinely archived changes exit 1 "No change matches". Implement the soft skip in the binary (exit 0, explicit already-archived notice) — see Assumptions #2.
- **Archive partial failure** (YAML + non-zero) is real behavior but undocumented → document it in `_cli-fab.md` (doc-side fix).
- `fab batch archive` exits 1 on empty sets while reporting `failed: 0` → exit 0 on an empty set (benign no-op).
- **Restore `--switch` swallows activation failure**: `src/go/fab/internal/archive/archive.go:171-178` renders a failed activation as if not requested → surface it (`pointer: failed`).

### 5. `stage_metrics` review-iterations survive the fail+reset choreography (Theme 4 item)

The rework choreography's reset cascade deletes `stage_metrics.review`, so PR meta always reports "1 cycle" — `SPEC-_pipeline.md:20`'s PR-meta rationale is currently false; the choreography zeroes the very counter it cites as payoff. Fix in Go: preserve the review `Iterations` counter across `fail`+`reset` (or derive it from `.history.jsonl` — see Assumptions #3). Update `SPEC-_pipeline.md` if the mechanism changes.

### 6. `fab hook sync` vs "All hook subcommands exit 0"

`fab hook sync` violates the `_cli-fab.md:168` claim. Listed under the binary-conformance actions: make `fab hook sync` exit 0 (errors surfaced on output, not exit code), consistent with the hook contract that protects agent flows — see Assumptions #4.

### 7. Doc pass (after the Go fixes)

- Fix `_preamble.md:232`: canonical form `fab change resolve --folder` is an invalid command (`ERROR: unknown flag: --folder`, exit 1; reproduced) — only top-level `fab resolve` registers the query flags. Replace with `fab resolve --folder`.
- Correct any `_preamble.md`/`_cli-fab.md` exit-code rows that remain *intentionally* exit-0 after the Go pass (i.e., behaviors deliberately not changed get docs-to-match-binary instead).
- Document archive partial-failure semantics (YAML + non-zero) in `_cli-fab.md`.
- `_preamble.md:230` / `_cli-fab.md:91` / `_pipeline.md` Pre-flight 3 need **no edit** — item 1 makes them true.

## Affected Memory

- `pipeline/schemas`: (modify) state machine gains AllowedStates enforcement on transitions; `fab score --check-gate` exit contract; `stage_metrics.review` survives fail+reset
- `pipeline/change-lifecycle`: (modify) `fab change switch` Next: routing aligned with fab-status; archive/batch-archive exit semantics
- `pipeline/execution-skills`: (modify) `/fab-archive` re-archive soft skip now real; restore `--switch` activation-failure surfacing

## Impact

- **Go** (`src/go/fab/`): `cmd/score.go` (or `cmd/fab/score.go` — locate at apply), `internal/status/status.go`, `internal/change/change.go`, `internal/archive/archive.go`, batch-archive command path, hook sync command path. Constitution: every Go change needs test updates (`**/*_test.go`, test-alongside) **and same-PR `src/kit/skills/_cli-fab.md` updates**.
- **Skills** (`src/kit/skills/` — canonical; never edit `.claude/skills/`): `_preamble.md`, `_cli-fab.md`, `fab-archive.md`, possibly `_pipeline.md`/`fab-switch.md` if their wording references the broken behavior. Each touched skill requires its `docs/specs/skills/SPEC-*.md` mirror update.
- **Sequencing**: wave 1 — ships first; w7dp (batch 2/5) branches after this merges; g8st/c5tr run parallel but coordinate the fab-archive exit-semantics seam (k4ge owns the Go/doc side of that contract).
- **Backlog marking at archive time**: the `[k4ge]` entry exists only in the main repo's `fab/backlog.md` — the eventual archive/backlog-mark step must run where that file is current.

## Open Questions

None — all decision points resolved as graded assumptions below (no Unresolved).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly the backlog k4ge ACTION list: Theme 1 findings + the Theme 4 stage-metrics item; the other 2026-06-12 audit themes belong to batches w7dp/g8st/c5tr/d9rs | Backlog entry enumerates actions with file:line refs and names the report sections; batch boundaries were designed for sequencing | S:95 R:90 A:95 D:90 |
| 2 | Confident | Re-archive: implement the documented soft skip in the binary (exit 0, already-archived notice) rather than rewriting docs to match the exit-1 behavior | Constitution VII (implementation conforms to spec) + III (idempotent re-runs); backlog lists it among binary-conformance actions; g8st separately wants idempotent backlog marking on re-run, which composes with this | S:75 R:80 A:85 D:75 |
| 3 | Tentative | Preserve `stage_metrics.review.Iterations` across the fail+reset cascade in Go (rather than deriving the cycle count from `.history.jsonl`) | Backlog offers both; preservation is the smaller, localized change and keeps `.status.yaml` self-contained, but history-derivation is more robust to future resets — apply may flip if the cascade code makes preservation awkward | S:60 R:75 A:60 D:50 |
| 4 | Confident | `fab hook sync`: make the binary exit 0 (conform to the "All hook subcommands exit 0" claim) rather than scoping the doc claim | Listed under binary-conformance actions (before "THEN the doc pass"); hook exit codes carry semantics in agent harnesses, so exit-0 protects flows; Constitution VII | S:70 R:85 A:75 D:70 |
| 5 | Confident | `fab change archive` no-arg → `ExactArgs(1)` usage error (non-zero); `fab batch archive` on empty set → exit 0 with `failed: 0` | Both named explicitly in the backlog entry; cobra-standard arg validation; empty-set is a benign no-op (idempotency) | S:85 R:85 A:85 D:85 |
| 6 | Certain | Findings embedded above are authoritative for this change; the full report (main repo, uncommitted) is consulted opportunistically at apply but is not a dependency | Intake is the state-transfer document; report inaccessible from this worktree by design | S:90 R:90 A:90 D:90 |

6 assumptions (2 certain, 3 confident, 1 tentative, 0 unresolved).
