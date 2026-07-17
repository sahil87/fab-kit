# Plan: Migrate fab batch switch to wt's Explicit --checkout Contract

**Change**: 260717-otol-batch-switch-explicit-checkout
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md's "What Changes" (7 subsections) + the SRAD assumptions.
     The change is a coordinated migration to the wt 2af2 contract: fab batch switch
     must probe branch existence and route to --checkout (existing) or the positional
     (new); docs/specs that teach the old dual-semantics invocation must be corrected. -->

### batch switch: Probe-and-route the wt create invocation

#### R1: Route existing branches through `--checkout`, new branches through the positional
`fab batch switch` SHALL probe whether the target branch already exists (local first, then remote-only) and route the `wt create` invocation accordingly, mirroring wt's own dispatch under the 2af2 contract where the positional is new-branch-only and `--checkout <branch>` is the explicit opt-in for an existing branch. `--reuse --worktree-name <match>` SHALL be retained on both routed forms.

- **GIVEN** a change whose branch (`branchPrefix + match`) already exists locally
- **WHEN** `fab batch switch <change>` runs
- **THEN** it invokes `wt create --non-interactive --reuse --worktree-name <match> --checkout <branch>` (no positional branch arg)

- **GIVEN** a change whose branch does not exist locally but exists on `origin`
- **WHEN** `fab batch switch <change>` runs
- **THEN** the remote probe (`git ls-remote --heads origin <branch>`) matches and the `--checkout` form is used

- **GIVEN** a change whose branch exists neither locally nor remotely
- **WHEN** `fab batch switch <change>` runs
- **THEN** it invokes the positional form `wt create --non-interactive --reuse --worktree-name <match> <branch>` (new-branch creation, unchanged from prior behavior)

#### R2: The branch-existence probe mirrors wt's `BranchExistsLocally`/`BranchExistsRemotely`
`fab batch switch` SHALL determine local existence via `git show-ref --verify --quiet refs/heads/<branch>` and, only when that fails, remote existence via `git ls-remote --heads origin <branch>` (origin-scoped, matching non-empty output). A failed or offline `ls-remote` SHALL degrade to not-remote (→ positional), so wt itself re-checks and errors visibly rather than fab guessing.

- **GIVEN** the local `show-ref` probe succeeds
- **WHEN** routing is decided
- **THEN** the remote `ls-remote` probe is NOT run (local-first, no unnecessary network call) and `--checkout` is chosen

- **GIVEN** `git ls-remote` fails (offline) or returns empty output
- **WHEN** routing is decided
- **THEN** the branch is treated as not-existing and the positional (new-branch) form is used

#### R3: Surface wt's stderr in the warn-and-skip line
`fab batch switch` SHALL invoke `wt create` via `pane.RunCmd` (capturing stdout and stderr separately) instead of `exec.Command(...).Output()` (which discards stderr), and on a `wt create` failure SHALL surface the child stderr via `pane.StderrError` in the warn-and-skip line so wt's typed exit-2 error and fix hint reach the operator. The failure remains warn-and-skip (loop continues; the command exit code is unchanged).

- **GIVEN** `wt create` exits non-zero (e.g., unknown-flag on an old wt, or an unexpected wt error)
- **WHEN** the failure is reported
- **THEN** the stderr line reads `Error: failed to create worktree for '<match>' (<err>: <wt-stderr>), skipping` and the loop continues to the next change

### batch new: audited, unchanged

#### R4: `fab batch new` is not modified
`fab batch new`'s `wt create` invocation passes no positional branch argument (the exploratory-create path), which wt 2af2 leaves unchanged. `batch_new.go` SHALL NOT be modified by this change.

- **GIVEN** the 2af2 contract ("`fab batch new` (no positional) is unaffected")
- **WHEN** this change is applied
- **THEN** `batch_new.go` has no diff

### Documentation: kit skills teach the routed contract

#### R5: `_cli-external.md` § wt documents the new branch-selection contract
`src/kit/skills/_cli-external.md` § wt SHALL rewrite the `[branch]` flags-table row to new-branch-only-plus-exit-2, add a `--checkout <branch>` row, rewrite the Known-change example and Operator Spawning Rules § Known change to probe-and-route, and drop `--base` from the `--checkout` arm of the autopilot-respawn example (`--checkout`+`--base` is a hard exit-2 conflict). The bare exploratory-create wording (§ New change) SHALL remain unchanged.

- **GIVEN** an operator reads `_cli-external.md` § wt for the spawn invocation
- **WHEN** the change is targeting an existing (known) change
- **THEN** the doc teaches: branch exists → `wt create --non-interactive --worktree-name <name> --checkout <change-folder-name>`; missing → the positional form

#### R6: `fab-operator.md` invocation lines reference the routed form
`src/kit/skills/fab-operator.md` SHALL update its literal `wt create ... <branch>` invocation lines (idea-lookup single-match action ~L110; §6 spawn-sequence step 2 ~L436; entry-form-table Existing-change parenthetical ~L539) to reference the routed form, delegating the flag detail to `_cli-external.md` § wt. The bare `wt create --non-interactive` (L35) SHALL remain unchanged.

- **GIVEN** an operator follows the §6 spawn sequence for an existing change
- **WHEN** it reads step 2
- **THEN** the `wt create` invocation reflects that an existing branch routes via `--checkout` (per `_cli-external.md` § wt)

#### R7: `_cli-fab.md` § fab batch switch bullet documents routing + stderr surfacing
`src/kit/skills/_cli-fab.md`'s `switch` bullet SHALL gain a routing sentence (probes branch existence — local `show-ref`, then `ls-remote --heads origin` — and passes `--checkout <branch>` for existing branches / the positional for new ones, per wt's 2af2 contract) and note that wt failures now surface the child stderr in the warn-and-skip line.

- **GIVEN** a reader of `_cli-fab.md` § fab batch
- **WHEN** they read the `switch` bullet
- **THEN** it describes the probe-and-route behavior and the stderr surfacing

### Specs: companions + the three SPEC mirrors

#### R8: `companions.md` § wt documents probe-and-route + minimum-wt coupling
`docs/specs/companions.md` § wt integration paragraph SHALL describe that `fab batch switch` probes branch existence and routes `wt create` to `--checkout`/positional, and note the minimum-wt version coupling (the `--checkout` path requires the wt release carrying 2af2).

- **GIVEN** a reader of `companions.md` § wt
- **WHEN** they read the `fab batch switch` integration sentence
- **THEN** it reflects the probe-and-route contract and the wt-version coupling note

#### R9: The three SPEC mirrors stay in sync with their skill sources
Per the constitution (every `src/kit/skills/*.md` edit updates its `docs/specs/skills/SPEC-*.md` mirror; treat the whole class), `SPEC-_cli-external.md`, `SPEC-fab-operator.md`, and `SPEC-_cli-fab.md` SHALL be updated to reflect the routed-contract changes in their source skills.

- **GIVEN** `_cli-external.md`, `fab-operator.md`, and `_cli-fab.md` are edited
- **WHEN** the change is reviewed
- **THEN** each corresponding SPEC mirror carries a matching update (inventory row / tools-table / decision entry as appropriate)

### Tests

#### R10: `batch_switch_test.go` asserts routing and stderr surfacing
Per the constitution (Go change ships tests), `src/go/fab/cmd/fab/batch_switch_test.go` SHALL extend the existing PATH-shim stub infra with an argv-capturing `wt` stub and a stubbed `git` controlling the probe, and assert: existing local branch → `--checkout` form; missing branch → positional form; probe failure → positional form; wt failure → stderr surfaced. Tests SHALL NEVER invoke the real installed `wt` binary (its old semantics differ from the migrated contract).

- **GIVEN** a stubbed `git` reporting the branch exists locally and an argv-capturing `wt`
- **WHEN** `runBatchSwitch` runs
- **THEN** the captured `wt` argv contains `--checkout <branch>` and no bare positional branch arg

- **GIVEN** a stubbed `git` reporting the branch missing (both probes fail) and an argv-capturing `wt`
- **WHEN** `runBatchSwitch` runs
- **THEN** the captured `wt` argv ends with the positional `<branch>` and contains no `--checkout`

- **GIVEN** a `wt` stub that exits non-zero writing a diagnostic to stderr
- **WHEN** `runBatchSwitch` runs
- **THEN** the warn-and-skip stderr line contains the child's diagnostic (via `pane.StderrError`) and the run exits without error

### Design Decisions

1. **Probe order local-then-remote, origin-scoped**: local `git show-ref --verify --quiet refs/heads/<b>` first (no network), remote `git ls-remote --heads origin <b>` only on local miss — *Why*: mirrors wt's own `BranchExistsLocally`/`BranchExistsRemotely` so fab's routing never disagrees with wt's positional validation; avoids an unnecessary network round-trip when the branch is local. *Rejected*: try-positional-then-retry-on-exit-2 (exit-code sniffing, two subprocess rounds, scary transient stderr); wt version detection/compat shim (contradicts the upstream hard-break decision).
2. **`pane.RunCmd` + `pane.StderrError` for stderr surfacing**: replace `.Output()` (stderr discarded) with the pattern `batch_new.go:118` already uses — *Why*: wt's typed exit-2 error carries the migration fix hint; the warn-and-skip line must carry it. Pattern reuse, not invention. *Rejected*: keeping `.Output()` and losing the stderr hint.
3. **No wt version detection**: migrated fab requires the wt release carrying 2af2; older wt fails the `--checkout` path loudly (unknown flag → warn-and-skip with stderr). *Why*: hard break was decided upstream; both tools share an author and release channel. *Rejected*: compat shim / version sniffing.

### Non-Goals

- Restructuring the broader stale-since-4rtx wt section in `kit-architecture.md` memory (hydrate corrects only the claims this change touches — a full wt-section cleanup is separate).
- Editing `docs/memory/` — that is hydrate's job, not apply's.
- Changing `batch_new.go` (audited unaffected).

## Tasks

### Phase 1: Core Implementation

- [x] T001 Add a `branchExists(branch string) bool` helper in `src/go/fab/cmd/fab/batch_switch.go` mirroring wt's probe: `git show-ref --verify --quiet refs/heads/<branch>` (local), else `git ls-remote --heads origin <branch>` (remote, non-empty output). Retain the `os/exec` import for the probe. <!-- R2 -->
- [x] T002 Rewrite the `wt create` invocation in `runBatchSwitch` (batch_switch.go:98) to build `wtArgs := []string{"create", "--non-interactive", "--reuse", "--worktree-name", match}` then append `--checkout <branchName>` when `branchExists(branchName)` else the positional `<branchName>`; invoke via `pane.RunCmd("wt", wtArgs...)` and on error surface `pane.StderrError(err, wtStderr)` in the warn-and-skip line. Add the `internal/pane` import; use `strings.TrimSpace(wtOut)` for the returned path. <!-- R1, R3 -->

### Phase 2: Tests

- [x] T003 Extend `src/go/fab/cmd/fab/batch_switch_test.go`: replace/augment the `wt` stub in `stubBatchSwitchTmuxCapture` (or add a new argv-capturing helper) so `wt`'s argv is captured to a file and `git` is stubbed to control the probe outcome; add tests asserting existing-local-branch → `--checkout` form, missing-branch → positional form, probe-failure → positional form, and wt-failure → stderr surfaced. Never invoke the real `wt`. <!-- R10 -->

### Phase 3: Documentation & Specs

- [x] T004 [P] Update `src/kit/skills/_cli-external.md` § wt: rewrite the `[branch]` flags-table row (new-branch-only + exit-2 note), add a `--checkout <branch>` row, rewrite the Known-change example (L120) and Operator Spawning Rules § Known change (L131-139) to probe-and-route, and drop `--base` from the `--checkout` arm of the autopilot-respawn example (L121). Leave § New change (bare create) unchanged. <!-- R5 -->
- [x] T005 [P] Update `src/kit/skills/fab-operator.md` invocation lines: idea-lookup single-match action (~L110), §6 spawn-sequence step 2 (~L436), and entry-form-table Existing-change parenthetical (~L539) to reference the routed form (delegating flag detail to `_cli-external.md` § wt). Leave the bare `wt create --non-interactive` (L35) unchanged. <!-- R6 -->
- [x] T006 [P] Update `src/kit/skills/_cli-fab.md` § fab batch `switch` bullet with the routing sentence (probe local `show-ref` then `ls-remote --heads origin`; `--checkout <branch>` for existing / positional for new, per wt's 2af2 contract) and the stderr-surfacing note. <!-- R7 -->
- [x] T007 [P] Update `docs/specs/companions.md` § wt integration paragraph: `fab batch switch` probes branch existence and routes `wt create` to `--checkout`/positional; add the minimum-wt coupling note (the `--checkout` path requires the wt release carrying 2af2). <!-- R8 -->
- [x] T008 [P] Update `docs/specs/skills/SPEC-_cli-external.md`: reflect the wt branch-selection contract change in the wt inventory row (and the mirror-rule note if warranted). <!-- R9 -->
- [x] T009 [P] Update `docs/specs/skills/SPEC-fab-operator.md`: reflect the routed `wt create` form in the wt-related content (tools-table row / spawn-sequence description). <!-- R9 -->
- [x] T010 [P] Update `docs/specs/skills/SPEC-_cli-fab.md`: reflect the batch-switch routing change in the `fab batch` inventory row. <!-- R9 -->

### Phase 4: Verify sweep + tests

- [x] T011 Repo-wide sweep across `src/kit/` and `docs/specs/`: grep the old dual-semantics claim (positional "create/checkout", "checked out in place", dual semantics) and confirm every occurrence in the mirror class is corrected; confirm the audited-unaffected bare-create mentions are untouched. <!-- R5, R6, R7, R8, R9 -->
- [x] T012 Run `cd src/go/fab && go test ./cmd/fab/...` and `gofmt -l` on the touched `.go` files; fix any failures. <!-- R1, R2, R3, R10 -->

## Execution Order

- T001 blocks T002 (T002 calls `branchExists`)
- T002 blocks T003 (tests exercise the new invocation) and T012
- T004-T010 are independent `[P]` documentation edits (different files)
- T011 runs after T004-T010; T012 runs after T002-T003

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab batch switch` routes existing branches through `--checkout <branch>` (no positional) and missing branches through the positional, retaining `--reuse --worktree-name <match>` on both forms.
- [x] A-002 R2: The probe uses local `git show-ref --verify --quiet refs/heads/<b>` first and remote `git ls-remote --heads origin <b>` only on local miss; a failed/offline remote probe degrades to not-remote.
- [x] A-003 R3: `wt create` runs via `pane.RunCmd` and a failure surfaces the child stderr via `pane.StderrError` in the warn-and-skip line (loop continues, exit unchanged).
- [x] A-004 R4: `batch_new.go` is unchanged (no diff).
- [x] A-005 R5: `_cli-external.md` § wt has the rewritten `[branch]` row, the new `--checkout` row, probe-and-route Known-change example + rule, and the `--base`-dropped `--checkout` respawn arm; § New change (bare create) is unchanged.
- [x] A-006 R6: `fab-operator.md`'s three invocation lines reference the routed form; the bare `wt create --non-interactive` (L35) is unchanged.
- [x] A-007 R7: `_cli-fab.md` § fab batch `switch` bullet carries the routing sentence and the stderr-surfacing note.
- [x] A-008 R8: `companions.md` § wt describes probe-and-route and the minimum-wt coupling note.
- [x] A-009 R9: All three SPEC mirrors (`SPEC-_cli-external.md`, `SPEC-fab-operator.md`, `SPEC-_cli-fab.md`) carry a matching update.

### Behavioral Correctness

- [x] A-010 R3: The warn-and-skip line format includes the wt stderr diagnostic (previously discarded by `.Output()`).
- [x] A-011 R1: The remote-only-branch case routes to `--checkout` (the exact shared-branch danger case 2af2 closes) rather than positional.

### Scenario Coverage

- [x] A-012 R10: `batch_switch_test.go` asserts all four cases (existing-local → `--checkout`; missing → positional; probe-fail → positional; wt-fail → stderr surfaced) using an argv-capturing `wt` stub and a stubbed `git`, never the real `wt` binary.

### Edge Cases & Error Handling

- [x] A-013 R2: Offline `ls-remote` (non-zero exit) degrades to positional so wt re-checks and errors visibly, not a silent skip.
- [x] A-014 R2: When the local probe succeeds, no `ls-remote` call is made (local-first, no unnecessary network).

### Code Quality

- [x] A-015 Pattern consistency: The `pane.RunCmd`/`pane.StderrError` usage matches `batch_new.go`'s existing pattern; the `branchExists` helper follows surrounding naming/error-handling style.
- [x] A-016 No unnecessary duplication: The probe reuses `os/exec` and `strings` already imported; stderr surfacing reuses the shared `internal/pane` helpers rather than reimplementing capture.
- [x] A-017 Canonical source only: All skill edits are in `src/kit/skills/`, none under `.claude/skills/`.
- [x] A-018 SPEC-mirror sync: Every edited `src/kit/skills/*.md` carries its `docs/specs/skills/SPEC-*.md` mirror update (whole-class sweep).
- [x] A-019 CLI ⇒ docs + tests: The `batch_switch.go` behavior change ships with `_cli-fab.md` updates and `batch_switch_test.go` test updates.
- [x] A-020 Go changes ship tests: The `.go` change is accompanied by test updates in the same change.

### Documentation Accuracy & Cross-References

- [x] A-021 Documentation accuracy: All updated docs describe the actual routed behavior implemented in `batch_switch.go` (no drift between prose and code).
- [x] A-022 Cross-references: `fab-operator.md`, `_cli-fab.md`, and `companions.md` remain consistent with `_cli-external.md` § wt (the single source of the flag detail); the audited-unaffected bare-create mentions are confirmed untouched.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (The retired `.Output()` invocation and its stderr-discarding warning line were replaced in place at `batch_switch.go:109-111`, not left behind; no other fab-kit code depended on the old dual-semantics positional.)

## Assumptions

<!-- Graded SRAD decisions made while co-generating ## Requirements. Three grades
     only (Certain/Confident/Tentative). These carry forward the intake's grounded
     assumptions plus apply-level under-specified points decided inline. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Probe-and-route mirroring wt's dispatch: local `show-ref --verify --quiet refs/heads/<b>`, else `ls-remote --heads origin <b>`; exists → `--checkout <branch>`, missing → positional | wt 2af2 intake explicitly anticipated this migration shape; mirroring `BranchExistsLocally/Remotely` keeps fab's routing in agreement with wt's validation | S:85 R:85 A:90 D:88 |
| 2 | Certain | `fab batch new` unchanged (no positional → exploratory create, contract unaffected) | 2af2 intake states it verbatim ("`fab batch new` (no positional) is unaffected") | S:90 R:95 A:95 D:95 |
| 3 | Certain | Keep `--reuse --worktree-name` on both routed forms | wt's reuse name-collision short-circuit ignores branch selectors, so routing is irrelevant when the worktree already exists | S:75 R:85 A:90 D:85 |
| 4 | Certain | `branchExists` is a package-level free function (not a method) taking the full branch name, placed in `batch_switch.go` next to `runBatchSwitch` | Matches the file's existing free-function layout (`listChanges`, `allChangeNames`); the probe needs no receiver state — smallest, most readable placement | S:80 R:88 A:88 D:85 |
| 5 | Confident | The documented autopilot-respawn example drops `--base` on the `--checkout` arm | `--checkout`+`--base` is a hard exit-2 conflict in new wt; `--base` is only meaningful for new branches | S:60 R:80 A:85 D:75 |
| 6 | Confident | batch_switch adopts `pane.RunCmd` + `pane.StderrError` (replacing `.Output()`'s stderr discard) | wt's typed exit-2 error is the designed migration signal; batch_new already uses this exact pattern — pattern reuse | S:55 R:85 A:85 D:70 |
| 7 | Confident | Remote probe scoped to `origin` only; failed/offline `ls-remote` degrades to not-remote → positional → wt re-checks and errors → visible warn-and-skip | Mirrors wt's own origin-only `BranchExistsRemotely`; degradation is loud, not silent | S:55 R:80 A:82 D:72 |
| 8 | Confident | The test stub captures `wt` argv to a file and stubs `git` to drive the probe deterministically, extending the existing PATH-shim infra; the real `wt` is never invoked | Installed wt (v0.0.23) still has OLD semantics; deterministic stubs are the only way to assert routing without version-coupled flakiness | S:60 R:82 A:85 D:78 |
| 9 | Confident | SPEC mirrors are updated as inventory-row / tools-table / decision-entry summaries (not verbatim skill prose) matching each mirror's existing catalog style | The three SPECs are catalog-style summaries, not line-by-line duplicates; the mirror rule requires a matching update, and matching the file's own granularity is the established convention | S:55 R:85 A:80 D:72 |

9 assumptions (4 certain, 5 confident, 0 tentative).
