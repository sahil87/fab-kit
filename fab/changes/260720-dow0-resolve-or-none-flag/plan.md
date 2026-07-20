# Plan: fab resolve --or-none — absence as a first-class query result

**Change**: 260720-dow0-resolve-or-none-flag
**Intake**: `intake.md`

## Requirements

### CLI: `fab resolve --or-none` (Go)

#### R1: Opt-in `--or-none` flag on `fab resolve`
`fab resolve` MUST accept a new boolean flag `--or-none` registered on the top-level `resolve` command only. The flag MUST NOT join the five-flag mutually-exclusive output-mode group — it composes with all of `--id`/`--folder`/`--dir`/`--status`/`--pane` and with `--server`. Without the flag, behavior MUST be byte-identical to today (absence-as-error; exit-code convention 0/1/2 untouched).

- **GIVEN** a repo with one resolvable change
- **WHEN** `fab resolve --folder --or-none <change>` runs
- **THEN** the output is the folder name, exit 0 — the flag is a no-op on the success path
- **AND** `fab resolve --status --folder` (no `--or-none` involved) still exits 2 as a flags-group conflict

#### R2: Sentinel mapping to `(none)` + exit 0
When `--or-none` is set and change resolution (`resolve.ToFolder`) fails, the CLI MUST map **state-sentinel** failures to a successful none-result: `errors.Is(err, resolve.ErrNotFound)` → `(none)` + exit 0 for **both** bare resolution and an explicit `<change>` override; `errors.Is(err, resolve.ErrAmbiguous)` → `(none)` + exit 0 **only** when no override argument was given (`changeArg == ""` — "multiple changes exist, none active" IS the no-active-change state). Ambiguous-with-override and infrastructure errors (`resolve.FabRoot()` failure, I/O) MUST stay non-zero, flag or no flag. `internal/resolve` is NOT changed — the sentinels already exist; this change only exposes them at the CLI surface.

- **GIVEN** a fab repo with no active change and no candidate changes
- **WHEN** `fab resolve --or-none` runs
- **THEN** stdout is exactly `(none)` and the exit code is 0
- **GIVEN** a fab repo with two changes both matching the override `add`
- **WHEN** `fab resolve --or-none add` runs
- **THEN** the `Multiple changes match …` error surfaces with a non-zero exit (unchanged)
- **GIVEN** a directory with no `fab/` root anywhere up the tree
- **WHEN** `fab resolve --or-none` runs
- **THEN** the `fab/ directory not found` error surfaces with a non-zero exit

#### R3: Output token exactly `(none)` across all output modes
The none-result token MUST be exactly `(none)` — not `none` (a legal 4-char change ID; collision risk with `--id` output) and not empty output (illegible in transcripts; hazardous in command substitution — `cd $(fab resolve --dir …)` with empty output cds to `$HOME`). `(none)` replaces the mode-specific output for every output mode. For `--pane`, the mapping applies to the **change-resolution** step only: a pane-lookup failure after successful resolution (`no tmux pane found …`, `not inside a tmux session`) is not a state sentinel and stays non-zero.

- **GIVEN** a fab repo with no active change
- **WHEN** `fab resolve --folder --or-none` and `fab resolve --id --or-none` run
- **THEN** both print exactly `(none)` with exit 0

#### R4: `fab change resolve` stays flag-free (sub-decision (b))
The `fab change resolve` thin wrapper MUST NOT gain the flag — the documented `_preamble.md` invariant "the query flags live on top-level `fab resolve` only" is preserved. `runResolve` stays the single shared implementation (the wrapper passes the flag as unset). Call sites needing the probe form migrate to top-level `fab resolve --folder --or-none` instead.

- **GIVEN** the built binary
- **WHEN** `fab change resolve --or-none x` runs
- **THEN** cobra rejects it with an unknown-flag usage error

### Go tests (test-alongside)

#### R5: Flag-path and regression test coverage
`src/go/fab/cmd/fab/resolve_test.go` MUST gain cases covering: bare not-found + `--or-none` → `(none)`/exit 0; explicit-override not-found + `--or-none` → `(none)`/exit 0; bare ambiguous + `--or-none` → `(none)`/exit 0; explicit-override ambiguous + `--or-none` → non-zero (message unchanged); infrastructure error (no `fab/` root) + `--or-none` → non-zero; flag absent → existing behavior byte-identical (bare and override not-found still error); output-mode composition (`--folder --or-none` and `--id --or-none` both emit `(none)` on the none path); success path with `--or-none` unchanged; `fab change resolve` rejects `--or-none`.

- **GIVEN** the new test cases
- **WHEN** `go test ./fab/cmd/fab/...` runs from `src/go`
- **THEN** all cases pass alongside the existing suite

### Kit skills: five probe-site migration

#### R6: Probe sites branch on the `(none)` token, not the exit code
Exactly the five sanctioned resolve-as-probe sites MUST migrate (canonical sources under `src/kit/skills/` only): `fab-discuss.md` Context Loading step 1 → `fab resolve --folder --or-none` (`(none)` ⇒ "No active change"); `fab-proceed.md` State Detection Step 1 → branch on `(none)` vs folder name; `_intake.md` backlog-ID probe → `fab resolve --id {id} --or-none` with the exact-ID equality compare preserved (`(none)` can never equal a 4-char ID); `fab-new.md` rename guard (context read + table rows 5/6) → `fab resolve --folder "$(git branch --show-current)" --or-none` branching `(none)` vs folder; `fab-adopt.md` collision guard → same form, a folder output (≠ `(none)`) ⇒ STOP. The `2>/dev/null` suppression is dropped at all five migrated sites (stderr then carries real-error guidance; the benign `(resolved from single active change)` note can only appear on the two bare-resolution sites, where surfacing it is harmless). `git-branch.md` (absence is a genuine STOP; its rename-guard twin keeps the strict form — divergence recorded in the fab-new sync comment) and `fab preflight` (the strict validation gate) MUST stay untouched.

- **GIVEN** the migrated skill sources
- **WHEN** grepping `src/kit/skills/` for `fab resolve` / `fab change resolve`
- **THEN** the five probe sites use `--or-none` token-branching; `git-branch.md`, `git-pr.md`, `git-pr-review.md`, and `fab-operator.md` retain their existing forms

### Reference docs

#### R7: `_cli-fab.md` + `_preamble.md` document the new surface
`_cli-fab.md` § fab resolve (extended) MUST document the flag: the sentinel mapping (not-found both forms / ambiguous bare-only / infrastructure never), the exact `(none)` token and its rationale, composition with the five output modes and `--server`, the pane-mode boundary, that `fab change resolve` remains flag-free, and the preflight-vs-resolve conceptual split (preflight = validation gate, non-zero by design; resolve = pure query that can answer "none" when asked). `_preamble.md` § Common fab Commands MUST update the `fab resolve` row: signature gains `[--or-none]`, purpose text gains the none-result semantics, canonical form becomes `fab resolve --folder --or-none`; the `fab change` row's flags-live-on-top-level note stays (verified still accurate).

- **GIVEN** the updated reference docs
- **WHEN** an agent reads either file
- **THEN** the flag's mapping, token, composition rules, and the preflight/resolve split are all discoverable

### Specs: mirror + aggregate sweep

#### R8: Whole SPEC-mirror class swept up front
Every touched skill's `docs/specs/skills/SPEC-*.md` mirror MUST be updated in the same change: `SPEC-fab-discuss.md`, `SPEC-fab-proceed.md`, `SPEC-_intake.md`, `SPEC-fab-new.md`, `SPEC-fab-adopt.md`, `SPEC-_cli-fab.md`, `SPEC-_preamble.md`. Aggregate restatements MUST be swept: `docs/specs/skills.md` (`/fab-proceed` state-detection step 1) and the shll skill bundle `docs/site/skill.md` (the Resolution bullet restates the resolve flag signature) with its build-time embedded copy `src/go/fab/cmd/fab/skill.md` re-synced (byte-identical, guarded by `TestSkillEmbedMatchesCanonical`). A final repo-wide re-grep of `fab resolve` (excluding `resolve-agent`) MUST confirm no stale probe-form restatement remains in the class.

- **GIVEN** the finished change
- **WHEN** `grep -rn "fab resolve" src/kit/skills/ docs/specs/ docs/site/ | grep -v resolve-agent` runs
- **THEN** every probe-form restatement of a migrated site reflects `--or-none`, and `TestSkillEmbedMatchesCanonical` passes

### Non-Goals

- No `internal/resolve` change — the sentinels and their `Unwrap` classification already exist
- No change to `fab preflight` (strict validation gate by design)
- No migration file — no user data (config, `.status.yaml`, archive layout) is restructured; binary + kit ship together per release, so flag and skill migration land atomically
- No migration of `git-branch.md`, `git-pr.md`, `git-pr-review.md`, `fab-operator.md` resolve call sites
- No distinct exit code for not-found, no change to flagless default behavior (rejected alternatives per intake)

### Design Decisions

#### Absence-as-data is strictly opt-in via `--or-none`
**Decision**: Map state-sentinel resolution failures to `(none)` + exit 0 only under a new opt-in flag; the flagless default stays absence-as-error.
**Why**: Fab's state machine treats "no active change" (`initialized`) as a first-class state, but the pure-query `fab resolve` could only fail on it — five sanctioned probe sites had to read the error channel as a data channel, rendering an expected state as a red error frame in Claude Code (alarm fatigue). Opt-in preserves the hard stop that `$(…)` consumers and `git-branch.md` rely on.
**Rejected**: skill-side `|| echo "(none)"` (repeats per site; swallows infrastructure errors); distinct exit code for not-found (any non-zero renders red; collides with exit 2 = usage error); changing the bare default to exit 0 + `(none)` (silently breaks `$(…)` consumers).
*Introduced by*: 260720-dow0-resolve-or-none-flag

#### The none token is exactly `(none)`
**Decision**: Print the literal token `(none)`, replacing the mode-specific output for every output mode.
**Why**: `none` is a legal 4-char change ID (collision with `--id` output); empty output is illegible to the primary consumer (an agent reading a transcript) and hazardous in command substitution (`cd $(fab resolve --dir …)` → `$HOME`).
**Rejected**: bare `none`; empty stdout.
*Introduced by*: 260720-dow0-resolve-or-none-flag

#### Sub-decision (b): `fab change resolve` stays flag-free
**Decision**: The fab-new/fab-adopt sites migrate to top-level `fab resolve --folder … --or-none`; the wrapper's surface does not widen.
**Why**: Preserves the documented `_preamble.md` invariant "the query flags live on top-level `fab resolve` only"; `runResolve` stays the single shared implementation.
**Rejected**: adding `--or-none` to `fab change resolve` (widens the wrapper surface and breaks the documented invariant).
*Introduced by*: 260720-dow0-resolve-or-none-flag

## Tasks

### Phase 1: Core Implementation (Go)

- [x] T001 Register the `--or-none` boolean flag on `resolveCmd` in `src/go/fab/cmd/fab/resolve.go` (outside the mutually-exclusive group; help text + probe example), thread an `orNone bool` parameter through `runResolve`, and map sentinel failures after `resolve.ToFolder` to a `noneToken = "(none)"` stdout line + nil error (`ErrNotFound` always; `ErrAmbiguous` only when `changeArg == ""`); `changeResolveCmd` in `src/go/fab/cmd/fab/change.go` passes `false` <!-- R1, R2, R3, R4 -->
- [x] T002 Extend `src/go/fab/cmd/fab/resolve_test.go` with the flag-path cases (bare/override not-found → `(none)`; bare ambiguous → `(none)`; override ambiguous → error; no-fab-root infra error; flagless regression; `--folder`/`--id` composition; success-path no-op; `fab change resolve` rejects the flag) and run `go test ./fab/cmd/fab/...` from `src/go` <!-- R5 -->

### Phase 2: Kit skill migration (canonical sources only)

- [x] T003 [P] Migrate `src/kit/skills/fab-discuss.md` Context Loading step 1 (and the Behavior step-2 restatement) to `fab resolve --folder --or-none` with `(none)`-token branching, dropping `2>/dev/null` <!-- R6 -->
- [x] T004 [P] Migrate `src/kit/skills/fab-proceed.md` State Detection Step 1 to `fab resolve --folder --or-none` — branch on `(none)` vs folder name; non-zero exit = real error (surface per the failure rule) <!-- R6 -->
- [x] T005 [P] Migrate `src/kit/skills/_intake.md` backlog-ID collision pre-check to `fab resolve --id {id} --or-none`, preserving the exact-ID equality compare (`(none)` never equals a 4-char ID) <!-- R6 -->
- [x] T006 [P] Migrate `src/kit/skills/fab-new.md` Step 11 rename guard: context read (line ~79) to `fab resolve --folder "$(git branch --show-current)" --or-none`, rows 5/6 condition text to `(none)`-vs-folder branching, and extend the git-branch.md keep-in-sync comment to record the deliberate probe-form divergence (git-branch keeps the strict form) <!-- R6 -->
- [x] T007 [P] Migrate `src/kit/skills/fab-adopt.md` Step 0 collision guard to `fab resolve --folder "$(git branch --show-current)" --or-none` — a folder output (≠ `(none)`) ⇒ STOP <!-- R6 -->
- [x] T008 Update `src/kit/skills/_cli-fab.md` § fab resolve (extended): signature gains `[--or-none]`, new flag-table row with the sentinel mapping + token + composition + pane boundary, prose for the token rationale, the `fab change resolve` flag-free invariant, and the preflight-vs-resolve conceptual split <!-- R7 -->
- [x] T009 Update `src/kit/skills/_preamble.md` § Common fab Commands: `fab resolve` row signature `[--or-none]`, purpose text with none-result semantics + preflight-split note, canonical form → `fab resolve --folder --or-none`; verify the `fab change` row note still reads correctly <!-- R7 -->

### Phase 3: Spec mirrors & aggregates

- [x] T010 [P] Update the seven SPEC mirrors in `docs/specs/skills/`: `SPEC-fab-discuss.md` (flow + tools), `SPEC-fab-proceed.md` (flow Step 1), `SPEC-_intake.md` (flow Step 3 + tools row), `SPEC-fab-new.md` (Step 11 context read + rename-guard wording), `SPEC-fab-adopt.md` (Step 0 collision-guard flow), `SPEC-_cli-fab.md` (fab resolve inventory row), `SPEC-_preamble.md` (Common-fab-Commands row description) <!-- R8 -->
- [x] T011 [P] Update the aggregate `docs/specs/skills.md` `/fab-proceed` Behavior item 1 (state-detection step 1 probe form) <!-- R8 -->
- [x] T012 Update `docs/site/skill.md` Resolution bullet with the `[--or-none]` signature + one-line semantics, re-sync the embedded copy `src/go/fab/cmd/fab/skill.md` (scripts/sync-skill.sh), and run the drift-guard test <!-- R8 -->
- [x] T013 Final sweep: re-grep `fab resolve`/`fab change resolve` (excluding `resolve-agent`) across `src/kit/skills/`, `docs/specs/`, `docs/site/` to confirm the class is fully swept and the untouched-by-design sites are intact; re-run `go test ./fab/cmd/fab/...` <!-- R8 -->

## Execution Order

- T001 blocks T002 (tests exercise the new flag) and T012 (embed drift-guard runs in the same package)
- Phase 2 and Phase 3 skill/spec edits are independent of the Go build but T010 depends on T003–T009 content being final (mirrors restate the skill text)
- T013 runs last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab resolve` accepts `--or-none` composing with all five output modes and `--server`; it is not in the mutually-exclusive group; flagless behavior is byte-identical — verified: `resolve.go:63` registers `--or-none` outside the `MarkFlagsMutuallyExclusive` group (line 64); composition + byte-identical flagless behavior covered by `TestResolveOrNoneOutputModeComposition` and `TestResolveWithoutOrNoneUnchanged`
- [x] A-002 R2: With `--or-none`, `ErrNotFound` (bare + override) and bare-only `ErrAmbiguous` map to `(none)` + exit 0; override-ambiguous and infrastructure errors stay non-zero; `internal/resolve` is untouched — `resolve.go:114-118` maps exactly this; `internal/resolve` absent from the diff; covered by `TestResolveOrNoneNotFound`/`BareAmbiguous`/`OverrideAmbiguousStillErrors`/`InfrastructureErrorStillErrors`
- [x] A-003 R3: The none token is exactly `(none)` and replaces the mode-specific output in every mode; pane-lookup failures after successful resolution stay non-zero — `noneToken = "(none)"` (`resolve.go:17`); the sentinel branch precedes the output-mode switch, and the pane path (line 132-133) is reachable only after successful resolution (line 112)
- [x] A-004 R4: `fab change resolve` rejects `--or-none` (flag-free wrapper preserved); `runResolve` remains the single shared implementation — `change.go:162` passes `false`; `TestChangeResolveHasNoOrNoneFlag` asserts the unknown-flag error
- [x] A-005 R6: All five probe sites (`fab-discuss`, `fab-proceed`, `_intake`, `fab-new`, `fab-adopt`) branch on the `(none)` token with `2>/dev/null` dropped; `git-branch.md` and `fab preflight` are untouched — verified in the diff; `git-branch.md:158` probe stays strict (only its sync comment updated); no `fab preflight` edit
- [x] A-006 R7: `_cli-fab.md` and `_preamble.md` document the flag surface, token, mapping, and the preflight-vs-resolve split — `_cli-fab.md` § fab resolve gains the flag row + token/split prose; `_preamble.md` Common-fab-Commands `fab resolve` row updated

### Behavioral Correctness

- [x] A-007 R2: Bare `fab resolve --or-none` in a multi-change/no-pointer repo prints `(none)` exit 0, while `fab resolve --or-none <ambiguous-override>` still errors with the unchanged `Multiple changes match` message — `TestResolveOrNoneBareAmbiguous` + `TestResolveOrNoneOverrideAmbiguousStillErrors` (message assertion `Multiple changes match`)
- [x] A-008 R6: The `_intake` backlog-ID equality compare still routes substring-slug matches away from resume (`(none)` and foreign-ID outputs both fail the equality check) — `_intake.md` preserves the exact-ID **equality** compare against `{id}`; `(none)` and a foreign canonical ID both fail equality

### Scenario Coverage

- [x] A-009 R5: Go tests cover all seven intake-listed scenarios (bare/override not-found, bare/override ambiguous, infra error, flagless regression, output-mode composition) and pass via `go test ./fab/cmd/fab/...` — all 9 new/updated tests pass (`go test ./cmd/fab/... -count=1` green from the `src/go/fab` module root; the plan's `src/go` cwd is a doc slip — the Go module is rooted at `src/go/fab`)

### Edge Cases & Error Handling

- [x] A-010 R2: Infrastructure failure (no `fab/` root) with `--or-none` still exits non-zero with the original error message — `TestResolveOrNoneInfrastructureErrorStillErrors` asserts non-nil error containing `fab/ directory not found` (skips gracefully if a `fab/` root exists above the temp dir)
- [x] A-011 R3: `--pane --or-none` with no resolvable change prints `(none)` exit 0 (mapping at the change-resolution step); pane failures after successful resolution are not mapped — verified by code structure: the sentinel map at `resolve.go:114` short-circuits before the `--pane` branch (line 132), which is reachable only on successful resolution; pane-lookup errors (`resolvePaneOutput`) are never sentinels
- [x] A-012 Pattern consistency: The flag registration, `mustBool` read, and `runResolve` threading follow the existing `resolve.go` structure; the token is a named constant (no magic strings) — `--or-none` registered alongside the other `cmd.Flags().Bool(...)` calls, read via the existing `mustBool` helper, threaded as `orNone bool`; token is the `noneToken` const
- [x] A-013 No unnecessary duplication: The sentinel classification reuses `errors.Is` against the existing `internal/resolve` sentinels — no parallel classification logic added — `errors.Is(err, resolve.ErrNotFound)` / `resolve.ErrAmbiguous` reused directly (`resolve.go:114-115`)
- [x] A-014 Canonical sources only: All skill edits are under `src/kit/skills/` (none under `.claude/skills/`) — diff confirms no `.claude/skills/` paths
- [x] A-015 SPEC-mirror sync (R8): All seven touched-skill SPEC mirrors plus `docs/specs/skills.md` and the `docs/site/skill.md` bundle (with byte-identical embedded copy) are updated in this change — all 7 mirrors + `SPEC-git-branch.md` (records the deliberate divergence) + `docs/specs/skills.md` + `docs/site/skill.md`/`src/go/fab/cmd/fab/skill.md` updated; `TestSkillEmbedMatchesCanonical` passes

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds an opt-in flag and migrates five probe sites to token-branching; it makes no existing code redundant. The `2>/dev/null` suppressions removed at the five migrated sites are already dropped in this diff (not leftover dead code), `internal/resolve` is untouched (the sentinels it exposes were already present), and `runResolve` stays the single shared implementation (no parallel path introduced).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `runResolve` gains an `orNone bool` parameter; `changeResolveCmd` passes `false` | The wrapper shares `runResolve` (single-implementation invariant), so the flag must thread through as a parameter the wrapper never sets — the only shape that preserves both the shared implementation and the flag-free wrapper surface | S:85 R:90 A:95 D:95 |
| 2 | Confident | `git-branch.md`'s Step 4 rename-guard twin keeps the strict `fab change resolve … 2>/dev/null` form despite the fab-new keep-in-sync comment; the divergence is recorded in that comment rather than migrating a sixth site | Intake assumption #4 (Certain) enumerates exactly five sites and states "`git-branch.md` is not migrated" flatly; the twin-sync comment already carries one deliberate divergence (dirty-count derivation), so a second recorded divergence follows the established pattern | S:75 R:90 A:85 D:70 |
| 3 | Confident | `2>/dev/null` dropped at all five migrated sites | Intake #7 delegated per-site; post-migration stderr at these sites carries only real-error guidance plus the benign `(resolved from single active change)` note, which can only fire on the two bare-resolution sites (fab-discuss, fab-proceed) where surfacing it is informative, not alarming | S:75 R:95 A:85 D:75 |
| 4 | Confident | The shll skill bundle (`docs/site/skill.md` Resolution bullet) is in the R8 sweep class, updated with the embedded copy re-synced | The bundle restates the `fab resolve` flag signature verbatim; constitution § Toolkit Standards binds CLI-surface changes to the `skill` standard (byte-identical embed, ≤150-line budget — the edit is additive within budget) | S:80 R:90 A:90 D:85 |
| 5 | Certain | The none token is a named constant (`noneToken`) in `resolve.go` | code-quality.md § Anti-Patterns bans magic strings; the token appears in code and is contract-bearing | S:70 R:95 A:95 D:95 |

5 assumptions (2 certain, 3 confident, 0 tentative).
