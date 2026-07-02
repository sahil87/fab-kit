# Plan: Pull-based artifact state — `fab status refresh` + self-healing transitions, drop the artifact-write hook

**Change**: 260702-y022-status-refresh-drop-artifact-hook
**Intake**: `intake.md`

## Requirements

### Refresh: recompute artifact-derived state

#### R1: `fab status refresh <change>` recomputes artifact-derived `.status.yaml` fields
The `fab status refresh <change>` command SHALL recompute the four artifact-derived
`.status.yaml` field groups — `change_type`, `confidence.*`/`confidence.score` (from `intake.md`),
and `plan.generated`/`plan.task_count`/`plan.acceptance_count`/`plan.acceptance_completed` (from
`plan.md`) — from on-disk artifacts, reusing the shared `Refresh` logic (R2). It SHALL resolve the
change like every other `fab status` subcommand (4-char ID / folder substring / full folder name),
acquire the `.status.yaml` flock via the existing `withStatusLock` path, load the `StatusFile` once,
mutate in memory, and Save exactly once only if dirty.

- **GIVEN** a change whose `intake.md` and `plan.md` are both present and whose `.status.yaml` derived fields are stale
- **WHEN** `fab status refresh <change>` runs
- **THEN** `change_type`, `confidence.*`, and the four `plan.*` counters are recomputed from the artifacts and persisted in a single Save

#### R2: `Refresh` is extracted from `artifactBookkeeping` into a shared internal package — no reimplementation
The recompute logic currently in `artifactBookkeeping` (`src/go/fab/cmd/fab/hook.go`, `main` package)
SHALL be extracted into a shared internal package (`internal/refresh`), exposing
`Refresh(fabRoot, changeDir string, sf *statusfile.StatusFile) (dirty bool, err error)`. The command
(R1) and the self-healing transitions (R4, R5) SHALL call this shared function; the logic SHALL NOT
be reimplemented anywhere. Reused helpers stay where they live (they have other live callers):
`hooklib.InferChangeType`, `hooklib.HasSectionHeading`, `hooklib.CountSectionItemsBounded`,
`hooklib.CountCompletedSectionItemsBounded`, `score.ComputeWithStatus`,
`status.ApplyChangeType`/`ApplyAcceptance`, and the `statusfile.SourceExplicit` machinery.

- **GIVEN** the shared package
- **WHEN** any of `refresh`/`advance`/`finish`/`preflight` needs to recompute artifact-derived state
- **THEN** it calls `refresh.Refresh` and no copy of the recompute logic exists elsewhere

#### R3: `Refresh` inspects BOTH artifacts on disk, preserves the `explicit` guard, tolerates missing artifacts, and is idempotent
`Refresh` SHALL inspect both `intake.md` AND `plan.md` under the change dir (not scoped to a single
written file as the hook's `match.Artifact` was). It SHALL recompute `change_type` (via
`hooklib.InferChangeType`) ONLY when `change_type_source` is absent or `inferred`, keeping the explicit
type and skipping inference when it is `explicit`. It SHALL recompute `confidence.*`/`score` via
`score.ComputeWithStatus` when `intake.md` is present. For `plan.md`, it SHALL set `plan.generated=true`
when `## Tasks` is present and recount `task_count`/`acceptance_count`/`acceptance_completed`, each
guarded by `HasSectionHeading` (a missing section leaves that field untouched — never zeroing a valid
value). It SHALL be a safe no-op when an artifact is absent (recomputing only what it can read), never
erroring on a missing artifact, and returning a non-nil error only on a genuine failure. It SHALL be
idempotent: running twice with unchanged artifacts produces the same `.status.yaml` and no spurious
`dirty` Save. The per-write `additionalContext` "Bookkeeping: ..." string SHALL be dropped (a
hook-only affordance no transition seam consumes).

- **GIVEN** a change with `change_type_source: explicit` and an intake whose prose would infer a different type
- **WHEN** `Refresh` runs
- **THEN** the explicit `change_type` is kept and not overwritten
- **AND GIVEN** a change with no `plan.md` yet (intake stage)
- **WHEN** `Refresh` runs
- **THEN** it recomputes only the intake-derived fields and does not error
- **AND GIVEN** unchanged artifacts
- **WHEN** `Refresh` runs a second time
- **THEN** `dirty` is false and no Save occurs

#### R4: Self-healing at `advance` and `finish`
`fab status advance` and `fab status finish` (`src/go/fab/cmd/fab/status.go`) SHALL run
`refresh.Refresh(fabRoot, changeDir, st)` inside their `withStatusLock` closure BEFORE the
`status.Advance`/`status.Finish` mutation, so the recompute and the transition persist in the same
locked load/Save. The `advance` closure (which currently discards `fabRoot` as `_ string`) SHALL
capture `fabRoot` and derive `changeDir` to call `Refresh`; `finish`'s closure already receives it.

- **GIVEN** a `.status.yaml` whose derived fields are stale after a hook-bypassing artifact edit
- **WHEN** `fab status advance` or `fab status finish` runs on that change
- **THEN** the derived fields are healed and the stage transition both persist in the single locked Save

#### R5: Self-healing at `preflight`
`fab preflight` (`src/go/fab/cmd/fab/preflight.go` → `internal/preflight`) SHALL run the same locked
refresh (via `lockfile.WithLock`) before its read-only derivation, moving preflight from pure-reader
to load-mutate-save. The read-only YAML derivation (formatting, `LiveAcceptance`) SHALL remain
unchanged; only the pre-read refresh is added. The refresh SHALL be best-effort — a refresh failure
SHALL NOT abort preflight's read/orient output (preflight must still surface state even if a recompute
step fails), consistent with preflight's role as the orient seam.

- **GIVEN** a change whose `.status.yaml` `change_type`/`confidence` are stale
- **WHEN** `fab preflight` runs
- **THEN** those fields are healed and persisted, and the emitted YAML reflects the healed values

#### R6: `start`/`reset`/`skip`/`fail` do NOT self-heal
`fab status start`, `reset`, `skip`, and `fail` SHALL NOT run `Refresh` — they move stage pointers
without a preceding artifact-generation write. The forward seams (`advance`/`finish`) plus the
orient seam (`preflight`) cover every reachable-stale window.

- **GIVEN** any of `start`/`reset`/`skip`/`fail`
- **WHEN** it runs
- **THEN** no refresh occurs and the surface stays minimal

### Hooks: remove artifact-write ownership

#### R7: Drop the `artifact-write` registration rows from both binaries' mapping tables
The two `artifact-write` PostToolUse rows (Write matcher, Edit matcher) SHALL be removed from
`hooklib.DefaultMappings` (`src/go/fab/internal/hooklib/sync.go`) so `fab hook sync` stops
(re-)registering them, and the `on-artifact-write.sh` entry SHALL be dropped from
`oldScriptToSubcommand`. The replicated table in the `fab-kit` binary
(`src/go/fab-kit/internal/hooksync.go`: `defaultHookMappings` + `oldScriptToSubcommand`) SHALL be
swept in lockstep.

- **GIVEN** a project with no artifact-write hook entries
- **WHEN** `fab hook sync` runs
- **THEN** it registers only the three session hooks (SessionStart/Stop/UserPromptSubmit) and no PostToolUse entry

#### R8: `fab hook artifact-write` becomes a one-release no-op shim
The `hookArtifactWriteCmd` subcommand and its `AddCommand` registration SHALL be retained for one
release as a no-op that writes nothing to stdout and exits 0. Rationale (apply-verified): an
unregistered `fab hook <x>` subcommand exits 0 but prints cobra help text (~505 bytes) to **stdout**,
which a still-registered PostToolUse hook on an un-migrated project feeds to Claude Code as
`additionalContext` (invalid JSON, mildly noisy). A silent no-op shim avoids that until the migration
(R9) removes the settings entry. The shim SHALL contain no bookkeeping logic (that moved to `Refresh`).

- **GIVEN** an un-migrated project whose `.claude/settings.local.json` still names `fab hook artifact-write`
- **WHEN** a Write/Edit fires that PostToolUse entry
- **THEN** the shim exits 0 and emits nothing on stdout (no noisy help output, no bookkeeping)

#### R9: Migration removes the artifact-write settings entries
A sentinel-guarded, idempotent migration `src/kit/migrations/2.10.1-to-2.11.0.md` SHALL remove the
two PostToolUse `fab hook artifact-write` entries (Write + Edit) from a project's
`.claude/settings.local.json`, leaving the three session-hook entries untouched. It SHALL bump
`src/kit/VERSION` 2.10.1 → 2.11.0. It SHALL follow the `settings.local.json`-editing precedent in
`0.46.0-to-1.1.0.md` §1 (read → parse `hooks.PostToolUse` array → drop entries whose `command` is
`fab hook artifact-write` for both matchers → write back atomically) and match the `2.9.2-to-2.10.0`
file format (`## Summary` / `## Pre-check` / `## Changes` / `## Verification`; atomic temp+rename
write; complete no-op on re-run with a `Skipped: ...already up to date` line when absent).

- **GIVEN** a project whose settings still list the two artifact-write PostToolUse entries
- **WHEN** the migration runs
- **THEN** those two entries are removed, the three session entries are preserved, and VERSION reads 2.11.0
- **AND WHEN** it re-runs
- **THEN** it is a complete no-op and prints the skip line

#### R10: Drop the hook's git-staging of `.status.yaml`/`.history.jsonl`
The best-effort `git add` of the change's `.status.yaml` and `.history.jsonl` (done today ONLY by the
artifact-write hook) SHALL be dropped, not relocated. Status/history files are committed at ship time
by `/git-pr`; the auto-stage was a convenience against transient "unstaged changes" friction, not a
correctness guarantee. It SHALL NOT be folded into `Refresh` (which stays a pure state-recompute) nor
into the transition commands.

- **GIVEN** the artifact-write hook removed
- **WHEN** artifacts are written and later a git operation runs
- **THEN** no automatic `git add` occurs; `/git-pr` stages status/history at ship as it already does

### Documentation, specs & memory sweep

#### R11: Sweep the full mirror class for hook-ownership and git-staging claims
Every load-bearing claim that the `artifact-write`/PostToolUse hook OWNS `change_type`, confidence, or
plan counts, or that it git-stages status/history, SHALL be rewritten in the same change to attribute
that recompute to `fab status refresh` (self-healed at the `advance`/`finish`/`preflight` seams),
across the full mirror class: every swept `src/kit/skills/*.md` and its `docs/specs/skills/SPEC-*.md`
mirror, the aggregate specs (`architecture.md`, `templates.md`, `change-types.md`,
`SPEC-hooks.md`), `_cli-fab.md` (+ its SPEC mirror), and the enumerated memory files. The `_cli-fab.md`
`fab status` family SHALL gain a `refresh` row; the four→three hook-count claims SHALL be corrected;
the `set-change-type` explicit-guard behavior SHALL be preserved (reworded to name `refresh`, not the
hook, as the recomputer).

- **GIVEN** the grep-enumerated surface (intake §7 + the apply reconnaissance sweep)
- **WHEN** the sweep completes
- **THEN** no canonical skill, SPEC mirror, aggregate spec, or `_cli-fab.md` still claims the hook owns those fields or git-stages, and each swept skill's SPEC mirror is updated in lockstep

### Tests

#### R12: Ship tests alongside all Go changes (Constitution VII)
All Go changes SHALL ship with tests in the same change. New `refresh`-package/command tests SHALL
cover: idempotency; respects `change_type_source: explicit`; intake-only present; plan-only present;
both present; missing-artifact no-op; missing-section leaves fields untouched. Self-healing tests SHALL
cover a sed-simulated stale `.status.yaml` healed by `advance`/`finish`/`preflight`. The
`artifact-write bookkeeping` tests in `hook_test.go` SHALL migrate to test the shared `refresh` path
(behavior unchanged; entry point changes) — with the shim's no-op behavior separately asserted (exit 0,
no stdout). `sync_test.go` and `hooksync_test.go` "expect 2 PostToolUse entries" assertions and the
`TestSyncHooks_ArtifactWriteDoubleMapping` test SHALL invert (expect the artifact-write rows absent).
`artifact_test.go` `InferChangeType`/`HasSectionHeading`/`CountSection*` tests stay (functions stay
live). `internal/status/status_test.go` `TestSetChangeType_MarksExplicit` stays; any doc comment
mentioning the hook is reworded.

- **GIVEN** the touched Go packages
- **WHEN** their tests run
- **THEN** they pass, and no test asserts the removed artifact-write registration or hook-owned bookkeeping

### Non-Goals

- Promoting the "hooks may enhance, never own" principle into spec/constitution PROSE — that is 3d's job (this change only states it in the intake and reattributes the swept docs).
- Any `.status.yaml` schema change — `change_type_source` already exists; refresh only changes *who* writes the existing derived fields.
- The 3b/3c/3d cross-harness dispatch machinery (per-tier `spawn_command`, `fab dispatch`, the CLI dispatch branch) — out of scope for 3a.
- Deleting reused helpers (`InferChangeType`, `HasSectionHeading`, `CountSection*`, `ComputeWithStatus`) — they retain live callers.

### Design Decisions

1. **Shared package home = new `internal/refresh`**: — *Why*: `artifactBookkeeping` currently lives in the `main` package (`cmd/fab/hook.go`) and cannot be imported by `internal/preflight` (import cycle / layering). A dedicated `internal/refresh` package is importable by `cmd/fab` (status/refresh commands) and `internal/preflight` alike, and keeps the recompute concern cohesive. — *Rejected*: folding into `internal/status` (already large; refresh composes `score` + `hooklib`, and `internal/status` importing `score` risks a cycle since `score` imports `status`).
2. **Delete-vs-shim → one-release no-op shim** (open call, apply-decided): — *Why*: apply-time check found an unregistered `fab hook <x>` exits 0 but prints help to stdout, which an un-migrated PostToolUse entry feeds to Claude Code as invalid `additionalContext` JSON (noisy). A silent no-op shim removes that noise for one release; the migration then removes the settings entry. — *Rejected*: delete-now (would leave un-migrated projects emitting noisy help text on every Write/Edit until they migrate).
3. **Git-staging → drop** (open call, apply-decided): — *Why*: the hook was the sole auto-stager; `/git-pr` commits status/history at ship. Folding into `Refresh` couples a pure recompute to git; folding into transitions adds git side effects to state mutations. — *Rejected*: fold-into-refresh / fold-into-transitions.
4. **`start` symmetry → no** (open call, apply-decided): — *Why*: `start` is not an artifact-generation seam; `advance`/`finish`/`preflight` cover every reachable-stale window. — *Rejected*: adding refresh to `start` (unnecessary surface).
5. **preflight refresh is best-effort**: — *Why*: preflight's contract is to orient (surface state); a recompute failure must not blind the caller. Advance/finish already heal on the forward path, so a preflight best-effort refresh is safe. — *Rejected*: hard-failing preflight on a refresh error (would break `fab preflight` on a transient issue).

## Tasks

### Phase 1: Extract shared Refresh

- [x] T001 Create `src/go/fab/internal/refresh/refresh.go` with `Refresh(fabRoot, changeDir string, sf *statusfile.StatusFile) (dirty bool, err error)` by moving the recompute logic out of `artifactBookkeeping` (`src/go/fab/cmd/fab/hook.go`): inspect BOTH `intake.md` and `plan.md` on disk under `changeDir`; preserve the `change_type_source: explicit` guard; recompute confidence via `score.ComputeWithStatus`; recount plan fields guarded by `HasSectionHeading`; tolerate missing artifacts (no error); return `dirty` and drop the `additionalContext` string. <!-- R2 R3 --> <!-- rework: cycle 1 — Refresh not dirty-idempotent: refresh.go:84-85 sets dirty=true unconditionally when intake.md exists; add compare-before-set so unchanged artifacts produce dirty=false (A-005). Also fix stale doc comments in score.go:183,:212,:344 that still name the artifact-write hook as a live ComputeWithStatus caller (should-fix). Do NOT remove the unused fabRoot param (R2 signature) -->
- [x] T002 Add `src/go/fab/internal/refresh/refresh_test.go` covering idempotency, explicit-guard, intake-only, plan-only, both-present, missing-artifact no-op, and missing-section-leaves-fields-untouched. <!-- R12 --> <!-- rework: cycle 1 — strengthen TestRefresh_Idempotent to assert dirty=false on the second run against unchanged artifacts (it currently checks value-stability only) (A-005) -->

### Phase 2: Command + self-healing wiring

- [x] T003 Add the `fab status refresh <change>` subcommand in `src/go/fab/cmd/fab/status.go` (register in `statusCmd`, route through `withStatusLock`, call `refresh.Refresh`, Save once if dirty) and thread its behavior through the existing single-load/single-Save discipline. <!-- R1 -->
- [x] T004 Wire self-healing into `fab status advance` and `fab status finish` (`src/go/fab/cmd/fab/status.go`): call `refresh.Refresh` inside the `withStatusLock` closure BEFORE the `Advance`/`Finish` mutation; capture `fabRoot` + derive `changeDir` in the `advance` closure (currently `_ string`). <!-- R4 -->
- [x] T005 Wire self-healing into `fab preflight` (`src/go/fab/cmd/fab/preflight.go`): run a locked `refresh.Refresh` (via `lockfile.WithLock`) best-effort before `preflight.Run`; keep the read-only derivation unchanged; a refresh error must not abort preflight output. <!-- R5 -->
- [x] T006 Add self-healing tests: a sed-simulated stale `.status.yaml` is healed by `advance`, by `finish`, and by `preflight` (in `src/go/fab/cmd/fab/status_test.go` / preflight test package as appropriate). Confirm `start`/`reset`/`skip`/`fail` do NOT refresh. <!-- R4 R5 R6 R12 -->

### Phase 3: Remove the hook

- [x] T007 Remove the two `artifact-write` rows from `hooklib.DefaultMappings` and the `on-artifact-write.sh` entry from `oldScriptToSubcommand` in `src/go/fab/internal/hooklib/sync.go`; sweep the replicated `defaultHookMappings` + `oldScriptToSubcommand` in `src/go/fab-kit/internal/hooksync.go` in lockstep. <!-- R7 -->
- [x] T008 Convert `hookArtifactWriteCmd` (`src/go/fab/cmd/fab/hook.go`) to a one-release no-op shim: keep the subcommand + its `AddCommand` registration, but its RunE writes nothing to stdout and exits 0 (no payload parse, no bookkeeping, no git-add). Delete the now-unused `artifactBookkeeping` and the hook's git-staging block. <!-- R8 R10 -->
- [x] T009 Update `src/go/fab/internal/hooklib/sync_test.go` and `src/go/fab-kit/internal/hooksync_test.go`: invert the "expect 2 PostToolUse entries" assertions to expect the artifact-write rows absent (0 PostToolUse entries), and drop/invert `TestSyncHooks_ArtifactWriteDoubleMapping`. Migrate the `TestHookArtifactWrite_*`/`runArtifactWriteHook` bookkeeping tests in `src/go/fab/cmd/fab/hook_test.go` to the `refresh` package (T002 covers behavior); replace with a shim test asserting exit 0 + empty stdout. Keep `artifact_test.go` `InferChangeType`/`HasSectionHeading`/`CountSection*` tests and `status_test.go` `TestSetChangeType_MarksExplicit` (reword any hook-mentioning doc comment). <!-- R12 -->

### Phase 4: Migration + doc/spec/memory sweep

- [x] T010 Author `src/kit/migrations/2.10.1-to-2.11.0.md` (sentinel-guarded, idempotent, atomic write) removing the two PostToolUse `fab hook artifact-write` entries from `.claude/settings.local.json`, preserving the three session entries, following `0.46.0-to-1.1.0.md` §1 + `2.9.2-to-2.10.0.md` format; bump `src/kit/VERSION` 2.10.1 → 2.11.0. <!-- R9 --> <!-- rework: cycle 1 — migration was authored at the wrong slot: file on disk is 2.10.0-to-2.11.0.md but pre-change VERSION was 2.10.1 (commit 72eb3cbf); rename file + heading to 2.10.1-to-2.11.0.md (content is correct) (A-004) -->
- [x] T011 Sweep `src/kit/skills/_cli-fab.md`: add a `refresh` row to the `fab status` family table; reword the `set-change-type` row (line ~115) so `refresh` (not the hook) is the recomputer that respects `explicit`; correct "the four event handlers" → three (line ~313); remove the `artifact-write` hook-table row (~320); remove/relocate the bookkeeping sub-section + git-staging sentence (~323–327). Mirror `SPEC-_cli-fab.md`. <!-- R11 --> <!-- rework: cycle 1 — _cli-fab.md:325 cites the migration as 2.10.0-to-2.11.0; correct to 2.10.1-to-2.11.0 after the T010 rename (A-004) -->
- [x] T012 Rewrite `docs/specs/skills/SPEC-hooks.md` substantially: `## Summary` four→three handlers; drop the `artifact-write` Registered-Hooks row; remove the `## artifact-write Bookkeeping` section + git-staging line + "the hook owns them mechanically"; correct the `## Event Coverage` PostToolUse row and the four→three count. Note the no-op shim + the migration. <!-- R11 --> <!-- rework: cycle 1 — SPEC-hooks.md:53 cites the migration as 2.10.0-to-2.11.0; correct to 2.10.1-to-2.11.0 after the T010 rename (A-004) -->
- [x] T013 Sweep the canonical skills + their SPEC mirrors: `_intake.md` Step 6 (+ `SPEC-_intake.md`), `_generation.md` plan-generation step (+ `SPEC-_generation.md`), `fab-continue.md` (+ `SPEC-fab-continue.md`), `fab-new.md` (+ `SPEC-fab-new.md`), and any `_pipeline.md`/`fab-ff.md` PostToolUse phrasing (+ `SPEC-_pipeline.md`, `SPEC-fab-ff.md`) — reattribute plan-count/change_type recompute to `fab status refresh` self-healed at the seams; preserve the sticky-explicit behavior. Use the apply reconnaissance sweep + intake §7 as the authoritative surface. <!-- R11 --> <!-- rework: cycle 1 — SPEC-_intake.md:5 Summary line still reads "6 (verify hook-owned `change_type`)"; sweep it (A-013). Optionally also drop the stale "◄── HOOK CANDIDATE (intake write)" annotation at SPEC-_intake.md:58 (nice-to-have) --> <!-- rework: cycle 2 — two residual stale "◄── HOOK CANDIDATE" annotations in SPEC-fab-continue.md:59 (Write: intake.md) and :72 (Write: plan.md) mark writes as artifact-write-hook fire points; drop both (or reword to "self-healed by fab status refresh at the advance/finish/preflight seams"). Then grep the whole SPEC mirror class for any remaining "HOOK CANDIDATE" markers tied to the removed hook (A-013/R11) -->
- [x] T014 Sweep the aggregate specs: `docs/specs/architecture.md` (~178), `docs/specs/templates.md` (~30/66/70/71), `docs/specs/change-types.md` (~84) — reattribute artifact-write-hook writes to `fab status refresh`. <!-- R11 -->
- [x] T015 Sweep the Affected-Memory files (all `(modify)`) per intake Affected Memory: `pipeline/change-lifecycle.md`, `pipeline/schemas.md`, `pipeline/planning-skills.md`, `pipeline/execution-skills.md`, `pipeline/index.md`, `runtime/runtime-agents.md`, `distribution/kit-architecture.md`, `distribution/setup.md`, `memory-docs/templates.md` — reattribute hook-owned state to `fab status refresh` + the transition seams; three session hooks only; per-domain index rows updated in lockstep. (Hydrate owns final memory writes; this task performs the sweep of load-bearing claims flagged as must-fix in intake §7.) <!-- R11 --> <!-- rework: cycle 1 — pipeline/index.md was never swept: lines 13 and 15 still claim change_type is hook-owned / hook re-infers (A-015). Also: kit-architecture.md:318 + runtime-agents.md:57 cite the migration as 2.10.0-to-2.11.0 — correct to 2.10.1-to-2.11.0 after the T010 rename (A-004); schemas.md:3 frontmatter description still says "hook re-infers only when not explicit" — reattribute to refresh (should-fix). Note: pipeline/index.md is generated by `fab memory-index` — fix the SOURCE frontmatter/description in the domain files if that's where the stale text lives, or regenerate via `fab memory-index`, rather than hand-editing the generated index -->

## Execution Order

- Phase 1 (T001–T002) blocks Phase 2 (the command and self-healing wiring import `refresh.Refresh`).
- T008 (delete `artifactBookkeeping`) MUST follow T001 (its logic must be moved out first).
- T009 (test migration/inversion) follows T007+T008.
- Phase 4 doc/migration/memory sweeps (T010–T015) are independent of the Go build and may run in parallel with each other, but T011–T015 should reflect the final Go behavior (shim + refresh + drop-git-staging) decided in Phases 1–3.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Deletion Candidates

- `fab hook artifact-write` shim + its `AddCommand` registration (`src/go/fab/cmd/fab/hook.go:188,39`) — intentionally retained for one release (R8); a genuine deletion candidate for the NEXT release once the 2.11.0 migration has removed the settings entries from deployed projects. Not deletable now (an un-migrated project's still-registered PostToolUse entry would print cobra help to stdout).
- `hooklib.ParsePayload`, `hooklib.MatchArtifactPath`, and the `ArtifactMatch` type (`src/go/fab/internal/hooklib/artifact.go:20,44,34`) — orphaned by this change (cycle-2 review finding): their sole production caller was the removed `artifactBookkeeping`; `refresh.Refresh` reads `intake.md`/`plan.md` directly and does no path-matching. Zero production callers remain in either binary (grep-verified; only their own tests reference them). Deletable alongside the shim next release; NOT covered by the Non-Goals "reused helpers" clause, which scoped itself to helpers with live callers.
- `score.ComputeWithStatus` (`src/go/fab/internal/score/score.go:249`) — NOT a deletion candidate; retains a live caller (`Compute`, the `fab score` path — score.go:221). Listed only to record it was checked (the refresh path moved to the new `ApplyToStatus`, but `ComputeWithStatus` is still used).
- `fabRoot` parameter of `refresh.Refresh` / `selfHealRefresh` (`src/go/fab/internal/refresh/refresh.go:52`, `src/go/fab/cmd/fab/status.go:336`) — unused since the confidence recompute switched to `score.ApplyToStatus` (which needs no fabRoot). Retained deliberately: R2 mandates the exact `Refresh(fabRoot, changeDir, sf)` signature. A candidate for removal only if R2's signature is relaxed in a later change; not deletable now without contradicting the plan.
- (Resolved) The cycle-1 misnamed `src/kit/migrations/2.10.0-to-2.11.0.md` was renamed to `2.10.1-to-2.11.0.md` and no longer exists; the prior deletion-candidate entry for it is retired.
- `artifactBookkeeping` and the hook's git-staging block — already deleted during apply (A-010 verified), so no longer candidates; recorded here for completeness (the deletion prompt's answer for what this change made redundant was acted on in-change, not deferred).

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab status refresh <change>` recomputes `change_type`, `confidence.*`, and the four `plan.*` fields from on-disk `intake.md`/`plan.md` and persists them in a single locked Save.
- [x] A-002 R2: The recompute logic lives once in `internal/refresh.Refresh`; `refresh`/`advance`/`finish`/`preflight` all call it, and no copy exists in `cmd/fab/hook.go`.
- [x] A-003 R7: `fab hook sync` (both binaries) registers only the three session hooks — no PostToolUse artifact-write entry.
- [x] A-004 R9: The `2.10.1-to-2.11.0.md` migration removes the two artifact-write PostToolUse entries, preserves the three session entries, and bumps VERSION to 2.11.0. MET (cycle-1 rework): the migration file on disk is `src/kit/migrations/2.10.1-to-2.11.0.md` (heading "Migration: 2.10.1 to 2.11.0"); the old `2.10.0-to-2.11.0.md` file no longer exists (grep-verified: zero `2.10.0-to-2.11.0` references remain in src/ or docs/). `src/kit/VERSION` reads 2.11.0. The Pre-check sentinel, the drop-both-matchers/preserve-custom-command logic, session-hook preservation, atomic write, and complete-no-op-on-re-run are all present. The four downstream doc refs (hook.go:180, sync.go:27, SPEC-hooks.md:53, _cli-fab.md:325, kit-architecture.md:318, runtime-agents.md:57) all cite the corrected slot.

### Behavioral Correctness

- [x] A-005 R3: `Refresh` keeps an `explicit` `change_type` (no re-inference), leaves a field untouched when its section heading is absent, is a no-op when an artifact is missing, and produces no spurious `dirty` Save on unchanged artifacts (idempotent). MET (cycle-1 rework): all three sub-paths now compare-before-set. Confidence: `refreshFromIntake` (refresh.go:93-97) snapshots `sfile.Confidence`, calls `score.ApplyToStatus`, and sets `dirty` only when `confidenceEqual` (which deep-compares the `*Dimensions` pointer, not pointer identity) reports a change. change_type (refresh.go:81-84) compares before/after the string. Plan block: `refreshFromPlan` (refresh.go:151,168) snapshots `sfile.Plan` and returns `sfile.Plan != before` (struct `!=`), sidestepping `ApplyAcceptance`'s always-mutate/always-nil-err. `TestRefresh_Idempotent` (refresh_test.go:299-305) now asserts `dirty=false` on the second run against unchanged artifacts — verified passing. Explicit-guard, missing-section, missing-artifact clauses all met and tested.
- [x] A-006 R4: A stale `.status.yaml` (hook-bypassing edit) is healed by `fab status advance` and `fab status finish`, with the heal and the transition in the same locked Save.
- [x] A-007 R5: A stale `.status.yaml` is healed by `fab preflight`; the read-only YAML derivation (incl. `LiveAcceptance`) is unchanged and a refresh failure does not abort preflight output.
- [x] A-008 R8: An un-migrated project's still-registered `fab hook artifact-write` invocation exits 0 and emits nothing on stdout (silent no-op shim; no bookkeeping, no git-add).
- [x] A-009 R10: No automatic `git add` of `.status.yaml`/`.history.jsonl` occurs anywhere on the status path; `/git-pr` remains the stager at ship.

### Removal Verification

- [x] A-010 R7: `artifactBookkeeping` and the hook git-staging block are gone from `cmd/fab/hook.go`; `on-artifact-write.sh` is gone from both `oldScriptToSubcommand` maps.

### Edge Cases & Error Handling

- [x] A-011 R6: `start`/`reset`/`skip`/`fail` do not run refresh (verified by test/inspection). `TestStart_DoesNotSelfHeal` + `TestReset_DoesNotSelfHeal` assert it for start/reset; skip/fail closures (status.go) contain no `refresh.Refresh`/`selfHealRefresh` call by inspection.
- [x] A-012 R3: `fab status refresh` exits non-zero only on genuine failure (unresolvable change, unreadable/corrupt `.status.yaml`), not on a missing artifact. `Refresh` returns `nil` error for missing artifacts (tested); the command surfaces errors only from `withStatusLock` resolve/`sf.Load` and `st.Save`.

### Documentation & Cross-References

- [x] A-013 R11: No canonical skill, SPEC mirror, aggregate spec, or `_cli-fab.md` still claims the artifact-write/PostToolUse hook owns `change_type`/confidence/plan counts or git-stages status/history; the recompute is attributed to `fab status refresh` self-healed at the seams. MET (cycle-1 rework): the previously-missed `SPEC-_intake.md:5` Summary line now reads "6 (verify `change_type` — recomputed by `fab status refresh`, self-healed at the transition seams)"; the Step 6 diagram (line 61) and command table (line 94) are consistent. Broad grep across `src/kit/skills/` + `docs/specs/` (canonical skills, all SPEC mirrors, architecture/templates/change-types/SPEC-hooks, `_cli-fab.md`) finds zero residual hook-ownership/git-stage claims. Remaining hits live only in `docs/specs/findings/*.md` (dated point-in-time review-finding archives, out of R11's sweep scope — they record what was true at that review).
- [x] A-014 R11: Every swept `src/kit/skills/*.md` has its `docs/specs/skills/SPEC-*.md` mirror updated in lockstep; `_cli-fab.md` gained the `refresh` row and its SPEC mirror is updated; the four→three hook-count claims are corrected everywhere. (Modified skills `_cli-fab`/`_generation`/`_intake`/`fab-continue` each have their SPEC mirror updated; `SPEC-_pipeline`/`SPEC-fab-ff`/`SPEC-fab-new`/`SPEC-hooks` updated for claims that lived only in specs — the canonical `_pipeline.md`/`fab-ff.md`/`fab-new.md` carry no such claim, verified by grep.)
- [x] A-015 R11: The enumerated Affected-Memory files reattribute hook-owned state to `fab status refresh` + the transition seams, with per-domain index rows updated in lockstep. MET (cycle-1 rework): the source `description:` frontmatter in `planning-skills.md` and `schemas.md` was corrected and `docs/memory/pipeline/index.md` was regenerated via `fab memory-index` — line 13 now reads "change_type is recomputed by `fab status refresh`, self-healed at the transition seams … (uliv; recompute moved off the PostToolUse hook in y022)" and "`fab status refresh` re-infers only when source is absent/inferred (jznd; hook replaced by pull-based refresh in y022)"; line 15 reads "`fab status refresh` re-infers only when not explicit — recompute moved off the PostToolUse hook to the pull-based refresh in y022". `fab memory-index --check --json` (worktree binary) reports `{tier:0, drift:false}` — the index is fresh. All eight domain files are swept correctly. Remaining hook-mention hits live only in `log.seed.md`/`log.md` (generated FKF changelog history of prior changes, e.g. 6bba which *added* the hook — out of R11's sweep scope). The new `pipeline/hooks-may-enhance-never-own.md` memory file is hydrate-owned per intake.

### Code Quality

- [x] A-016 Pattern consistency: New code follows the surrounding Go patterns — `withStatusLock`/`lockfile.WithLock` load-mutate-save discipline, `internal/` package layering, and the existing error-handling style. (`statusRefreshCmd` routes through `withStatusLock`; `refreshPreflightState` uses `lockfile.WithLock` best-effort; `internal/refresh` layering avoids the status↔score cycle.)
- [x] A-017 No unnecessary duplication: `Refresh` reuses the existing `hooklib`/`score`/`status` helpers rather than reimplementing inference, counting, or scoring. (Calls `hooklib.InferChangeType`/`HasSectionHeading`/`CountSectionItemsBounded`/`CountCompletedSectionItemsBounded`, `score.ApplyToStatus`, `status.ApplyChangeType`/`ApplyAcceptance`.)
- [x] A-018 No god functions / magic strings: extracted `Refresh` stays focused; artifact filenames/section names reuse existing named constants. (`Refresh` split into `refreshFromIntake`/`refreshFromPlan`; section names use `hooklib.SectionTasks`/`SectionAcceptance`. Minor: literal `"intake.md"`/`"plan.md"` are inline string literals rather than named constants — see should-fix.)

### Tests

- [x] A-019 R12: All touched Go packages' tests pass, including the migrated `refresh` tests, the self-healing tests, the shim no-op test, and the inverted sync/hooksync PostToolUse-count assertions. (fab: refresh 8, cmd/fab 219, hooklib 46, score 31, preflight 6, status 71 — 0 failures; fab-kit internal pass incl. `TestSyncHooks_NoArtifactWriteRegistration`. Both binaries build; `go vet` clean.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Shared package home is a new `internal/refresh` (not folded into `internal/status`) | `artifactBookkeeping` in `main` can't be imported by `internal/preflight`; `internal/refresh` is importable by both and avoids a `status`↔`score` import cycle (score imports status). Reversible if a cycle surfaces | S:75 R:70 A:80 D:70 |
| 2 | Confident | `Refresh` inspects BOTH intake.md and plan.md on disk and drops the hook-only `additionalContext` string | Follows directly from the transition-seam use (not scoped to one written file); no transition consumes additionalContext; one obvious shape, low blast radius (intake #12) | S:70 R:75 A:80 D:75 |
| 3 | Confident | `preflight` moves from pure-reader to locked load-mutate-save, best-effort (refresh error does not abort output) | Verified preflight is currently a pure reader; a locked pre-read refresh is the minimal change; making it best-effort protects preflight's orient contract (intake #13) | S:75 R:70 A:80 D:70 |
| 4 | Confident | Do NOT self-heal `start`/`reset`/`skip`/`fail` — only `advance`/`finish`/`preflight` | Not artifact-generation seams; forward+orient seams cover every reachable-stale window; minimal-surface principle (intake #14, open call → resolved: no) | S:70 R:75 A:80 D:70 |
| 5 | Confident | Git-staging of `.status.yaml`/`.history.jsonl`: DROP it (not fold into refresh or transitions) | Hook is the sole auto-stager; `/git-pr` stages at ship; folding couples a pure recompute (or a state transition) to git side effects — an anti-pattern. Reversible either way (intake #15, open call → resolved: drop) | S:60 R:60 A:70 D:65 |
| 6 | Confident | Keep `fab hook artifact-write` as a one-release no-op shim (silent exit-0, no stdout) rather than delete now | Apply-verified: an unregistered `fab hook <x>` exits 0 but prints ~505B of cobra help to STDOUT, which an un-migrated PostToolUse entry feeds Claude Code as invalid additionalContext JSON. A silent shim removes that noise until the migration lands (intake #16, open call → resolved: shim, on new evidence) | S:70 R:70 A:80 D:70 |
| 7 | Certain | Migration slot is `2.10.1-to-2.11.0.md` bumping VERSION 2.10.1 → 2.11.0 (NOT `2.10.0-to-2.11.0` as the intake assumed) | Verified `src/kit/VERSION`=2.10.1 (commit 72eb3cbf `release: v2.10.1`); last migration file is `2.9.2-to-2.10.0.md` and 2.10.1 shipped with no migration, so the next slot starts at the current version 2.10.1 | S:90 R:80 A:100 D:95 |
| 8 | Confident | Migrate the hook_test.go bookkeeping tests to the `refresh` package and add a separate shim no-op test; invert the sync/hooksync PostToolUse-count assertions (expect 0 artifact-write rows) | Behavior is preserved, only the entry point changes; the test surface is grep-enumerated and follows the code move mechanically (intake #11) | S:80 R:70 A:90 D:80 |
| 9 | Certain | Rework cycle 1: dirty-idempotency requires compare-before-set on ALL three recompute sub-paths (confidence, change_type, plan block), not just the confidence path the A-005 finding named | The A-005 finding cited only `refreshFromIntake` confidence (refresh.go:84-85), but the change_type-inference path and the plan-count path (`ApplyAcceptance` always mutates + returns nil) also flagged dirty unconditionally, so fixing confidence alone still re-Saves a plan-present change on every run. Fixed all three: confidence via `confidenceEqual` (deep-compares the fuzzy `Dimensions` pointer), change_type via before/after string compare, plan block via a `Plan` struct `!=` on the pre/post snapshot. Verified by the strengthened `TestRefresh_Idempotent` asserting dirty=false on the second run | S:90 R:80 A:95 D:90 |

9 assumptions (2 certain, 7 confident, 0 tentative).
