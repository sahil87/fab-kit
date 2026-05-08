# Plan: Restrain AI-driven code bloat

**Change**: 260507-ogf2-restrain-ai-code-bloat
**Status**: In Progress
**Intake**: `intake.md`
**Spec**: `spec.md`

## Tasks

### Phase 1: Setup

- [x] T001 [P] Add `## Parsimony Pass` section with single `Enabled: true` toggle to `src/kit/scaffold/fab/project/code-review.md` (the only project-level knob; thresholds and skip list are NOT exposed)
- [x] T002 [P] Document `## Deletion Candidates` section in the guidance comment at the top of `src/kit/templates/plan.md` (review-owned parser-contract heading; appended below `## Notes`; omitted entirely for `docs`/`chore`/`ci` change types). Do NOT add a placeholder section — review writes it lazily.
- [x] T003 [P] Add an `excludes` comment to `src/kit/templates/status.yaml` documenting that `true_impact` is created lazily on first apply-finish; do NOT add an empty placeholder.

### Phase 2: Core Implementation

- [x] T004 Create new Go package `src/go/fab/internal/impact/impact.go` containing the canonical shortstat math: merge-base resolution helper, `git diff --shortstat <base>...HEAD` parsing (insertions/deletions regex), pathspec-exclude assembly, and a `Compute(base, head string, excludes []string) (Result, error)` entry point that returns `{added, deleted, net}` plus optional `{excluding: {added, deleted, net}}`. Read `true_impact_exclude` via the existing `internal/config` package (extend if needed).
- [x] T005 Add `Compute` unit tests in `src/go/fab/internal/impact/impact_test.go` covering: (a) shortstat parsing happy path, (b) shortstat with only insertions or only deletions, (c) empty excludes returns no `excluding` block, (d) merge-base failure returns actionable error.
- [x] T006 Extend `src/go/fab/internal/config/config.go` to load the optional top-level `TrueImpactExclude []string` field (omitempty). Keep backwards compat: absent/null/empty all yield empty slice.
- [x] T007 Add `Compute` calls a thin orchestration helper that accepts a `fabRoot` and reads excludes via `config.Load`, then calls `impact.Compute`. Place this in `src/go/fab/internal/impact/impact.go` as `ComputeForRepo(fabRoot, base, head string) (Result, error)`.
- [x] T008 Create `src/go/fab/cmd/fab/impact.go` registering `fab impact <base> <head>` subcommand (cobra). Behavior: invoke `impact.ComputeForRepo`, marshal a YAML doc to stdout (`added`, `deleted`, `net`, optional `excluding`, `computed_at`), exit non-zero with an actionable stderr message on merge-base failure or `git diff` failure.
- [x] T009 Register the new `impactCmd()` in `src/go/fab/cmd/fab/main.go`'s root `AddCommand(...)` list.
- [x] T010 Add YAML output integration test in `src/go/fab/cmd/fab/impact_test.go`: invoke against a synthetic git repo, assert YAML schema (presence/absence of `excluding`).
- [x] T011 Extend `src/go/fab/internal/statusfile/statusfile.go` `StatusFile` to carry an optional `TrueImpact *TrueImpact` field (struct with `Added`, `Deleted`, `Net`, optional `Excluding *TrueImpactExcluding`, `ComputedAt`, `ComputedAtStage`). Wire load (parse `true_impact:` mapping when present) and save (emit block when non-nil; omit `excluding` sub-block when nil). Backwards compatible — absent block is fine.
- [x] T012 Add `WriteTrueImpact(statusPath, base, head, stage string)` helper in `src/go/fab/internal/status/true_impact.go` (new file) that calls `impact.ComputeForRepo`, attaches `computed_at_stage`, and saves the block via the existing `Save` flow. On computation failure: log a one-line warning to stderr and return nil (best-effort per spec assumption #17).
- [x] T013 Hook the helper into `status.Finish` (`src/go/fab/internal/status/status.go`) for stages `apply` and `hydrate` only — invoked AFTER `applyMetricsSideEffect` and the file save, BEFORE post-hooks. Best-effort: never propagate the error.
- [x] T014 Add unit tests in `src/go/fab/internal/status/true_impact_test.go` covering: (a) apply-finish writes block, (b) hydrate-finish overwrites with `computed_at_stage: hydrate`, (c) merge-base failure leaves `.status.yaml` unchanged + emits stderr warning, (d) empty `true_impact_exclude` omits `excluding` sub-block.

### Phase 3: Integration & Edge Cases

- [x] T015 Refactor `src/kit/skills/git-pr.md` Step 3c-impact: replace the inline `yq` + two `git diff --shortstat` invocations + parsing with a single `fab impact $BASE HEAD` invocation (parsing the YAML output for the rendering logic). Preserve the existing PR-body rendering (the `**Impact**: +A/−D code (excluding ...) · +A_total/−D_total total` line, the `+0/−0` omission rule, no-fab-context fallback). Net deletions expected.
- [x] T016 Add the parsimony pass + deletion-candidate prompt to `src/kit/skills/_review.md`'s Inward Sub-Agent. Specifically: (a) add Validation Step 7 "Parsimony pass" with the four-category table (`reuse-existing-utility`/Should-fix, `zero-call-sites`/Must-fix, `duplicated-logic`/Must-fix, `verbosity`/Nice-to-have), the hard-coded 100-line threshold (advisory), and the hard-coded skip list (`docs`, `chore`, `ci`); (b) add Validation Step 8 "Deletion-candidate prompt" emitting a `## Deletion Candidates` section appended to `plan.md` (replaces existing section on rework, omitted entirely for skip-list change types); (c) document the toggle: when `fab/project/code-review.md` `## Parsimony Pass` `Enabled` is `false`, skip the parsimony step (treat as skip-list match).
- [x] T017 Update `src/kit/skills/fab-status.md` to render an `Impact:` line under the change summary (sourced from `.status.yaml` `true_impact`); yellow-highlight when raw `net > 100` OR `excluding.net > 50`. Add a soft-warning line below the impact line when `change_type == refactor` AND (`excluding.net > 50` if present, else `net > 50`). Both thresholds hard-coded; no config knobs.
- [x] T018 Update `src/kit/skills/fab-continue.md` Hydrate Behavior Steps to read `## Deletion Candidates` from `plan.md` when present (informational only — hydrate MAY reference candidates in memory updates but MUST NOT modify the section). Document that absent section is treated as "no findings", no error.
- [x] T019 Add `--show-stats` flag to `fab change list` in `src/go/fab/cmd/fab/change.go` and `src/go/fab/internal/change/change.go`: when set, append `excluding.net` (or `net` if no excluding, or `—` if no block) as a new column to the `:`-delimited output. Default behavior unchanged.
- [x] T020 Add tests in `src/go/fab/internal/change/change_test.go` for `--show-stats` flag emitting the impact column correctly across (a) block present with excluding, (b) block present without excluding, (c) block absent.

### Phase 4: Polish

- [x] T021 [P] Document the `fab impact <base> <head>` subcommand in `src/kit/skills/_cli-fab.md` (add a new section after `fab kit-path`, before `fab fab-help`). Include argument format, YAML output schema, exit codes.
- [x] T022 [P] Create `docs/specs/skills/SPEC-_review.md` documenting `_review.md`'s flow including the new Parsimony Pass + Deletion Candidates validation steps. Mirror the structure of `SPEC-preamble.md` (Summary, Flow, Tools used, Sub-agents).
- [x] T023 [P] Update `docs/specs/skills/SPEC-fab-status.md` Flow to reflect the new Impact line + refactor warning rendering (sourced from `.status.yaml` `true_impact`).
- [x] T024 [P] Update `docs/specs/skills/SPEC-git-pr.md` Step 3c flow: replace the inline `git merge-base` + two `git diff --shortstat` invocations with a single `fab impact` invocation. Update the Tools used table.

## Execution Order

- T001-T003 are independent (different files, all `[P]`).
- T004-T010 are the impact package + CLI; sequential dependency: T004 → T005 → T006 → T007 → T008 → T009 → T010.
- T011-T014 are statusfile + finish-hook wiring; T011 → T012 → T013 → T014.
- Phase 3 depends on Phase 2 binary work being complete (T015 needs `fab impact`; T013 needs the helper; T017 needs the schema).
- Phase 4 docs are all `[P]` after Phase 3.

## Acceptance

### Functional Completeness

- [x] A-001 Parsimony Pass Added to Inward Sub-Agent: `_review.md`'s inward sub-agent runs a parsimony validation step alongside the existing six checks; emits findings classified into the four-category table with mapped severity (`reuse-existing-utility`/Should-fix, `zero-call-sites`/Must-fix, `duplicated-logic`/Must-fix, `verbosity`/Nice-to-have); findings cite specific file paths and line ranges.
- [x] A-002 Parsimony Threshold (Advisory, Hard-Coded): the 100-net-added-lines threshold is hard-coded in `_review.md` as a constant in the prompt body (not project-configurable); below threshold the pass still runs and may emit findings; above threshold the agent applies stricter scrutiny but no new severity tier.
- [x] A-003 Parsimony Skip List (Hard-Coded): `_review.md` skips the parsimony step when `change_type` is `docs`, `chore`, or `ci`; the skip list is hard-coded; other validation checks still run.
- [x] A-004 Parsimony Toggle in `code-review.md`: `## Parsimony Pass` section with `Enabled: true` is the only project-level knob; absent/missing/`true` runs the pass; `false` silently skips it.
- [x] A-005 Deletion-Candidate Prompt at Review: `_review.md`'s inward sub-agent produces a structured list of candidates (each naming a specific symbol/file path/block with one-line justification) or the literal "None — this change adds new functionality without making existing code redundant"; runs in review stage (not hydrate); never auto-deletes.
- [x] A-006 Deletion Candidates Section in `plan.md`: `_review.md` appends `## Deletion Candidates` below `## Notes` (or end of file if absent); section is replaced (not duplicated) on rework; section is distinct from `## Acceptance > ### Removal Verification`.
- [x] A-007 Deletion-Candidate Skip List: `_review.md` omits the section entirely (not even "None") when `change_type` is `docs`, `chore`, or `ci`.
- [x] A-008 Hydrate Reads Deletion Candidates When Present: `fab-continue.md` Hydrate Behavior reads the section when present and treats absence as "no findings" without error; never generates or modifies the section.
- [x] A-009 `plan.md` Template Documents the Section: `src/kit/templates/plan.md` describes (in a guidance comment) the `## Deletion Candidates` section as review-owned and review-generated; no placeholder section in the scaffolded output.
- [x] A-010 `true_impact` Block Added to `.status.yaml`: top-level optional block with fields `added`/`deleted`/`net`/`computed_at`/`computed_at_stage` plus optional `excluding {added, deleted, net}`; `excluding` omitted when `true_impact_exclude` is absent/null/empty; backwards-compatible (absent block is valid).
- [x] A-011 `fab impact` Go Subcommand: emits a YAML document to stdout with the `true_impact` schema (minus `computed_at_stage`); exits non-zero on merge-base/git-diff failure with an actionable stderr message; reads `true_impact_exclude` from `fab/project/config.yaml`.
- [x] A-012 `git-pr.md` Step 3c-impact consumes `fab impact`: the inline `yq` + two `git diff --shortstat` invocations + parsing are replaced by a single `fab impact $BASE HEAD` call; PR-body rendering is preserved (the Impact line format, the `+0/−0` omission rule, the no-fab-context fallback).
- [x] A-013 Computation Hook at Apply-Finish and Hydrate-Finish: `fab status finish <change> apply` and `fab status finish <change> hydrate` invoke the helper and write the block with the corresponding `computed_at_stage`. Best-effort: failure emits a one-line stderr warning and the stage transition still completes.
- [x] A-014 `/fab-status` Surfaces `true_impact`: renders an `Impact:` line under the change summary when the block is present; omits the line when absent; yellow-highlights when raw `net > 100` OR `excluding.net > 50`.
- [x] A-015 `fab change list --show-stats` Flag: adds an optional column showing `excluding.net` (or `net` if no excluding, or `—` if absent); default `fab change list` output unchanged.
- [x] A-016 Refactor-Growth Soft Warning: `/fab-status` emits the literal warning line `Refactor changes typically shrink or stay flat — review whether this growth is intentional.` when `change_type == refactor` AND threshold exceeded (excluding.net > 50 or net > 50); informational only.
- [x] A-017 SPEC Files Updated for Touched Skills: `SPEC-_review.md` (new), `SPEC-fab-status.md`, `SPEC-git-pr.md` are updated to reflect skill changes; `_cli-fab.md` documents the `fab impact` subcommand signature.

### Behavioral Correctness

- [x] A-018 Both `git-pr` Step 3c-impact and the apply-finish hook share `fab impact` and observe identical numbers for the same merge-base + HEAD.
- [x] A-019 Hydrate-finish recomputation overwrites the apply-finish block; `computed_at` advances, `computed_at_stage` becomes `hydrate`.
- [x] A-020 The parsimony pass on a `chore`-typed change is silently skipped without emitting findings; other validation checks still run.

### Removal Verification

- [x] A-021 The inline `yq` + two `git diff --shortstat` invocations + parsing are removed from `src/kit/skills/git-pr.md` Step 3c-impact (replaced by `fab impact` invocation). Net markdown deletion in git-pr.md.

### Scenario Coverage

- [x] A-022 Scenario: Reused-utility candidate flagged as Should-fix — verified by review-time prompt structure in `_review.md`.
- [x] A-023 Scenario: Zero-call-site code flagged as Must-fix — same as A-022.
- [x] A-024 Scenario: Apply-finish on broken worktree (no merge-base) — `.status.yaml` does NOT gain a `true_impact` block; stage transition still completes; covered by `true_impact_test.go`.
- [x] A-025 Scenario: Empty exclude list omits `excluding` sub-block — covered by `impact_test.go` and `true_impact_test.go`.
- [x] A-026 Scenario: First review cycle appends `## Deletion Candidates` after `## Notes`; rework cycle replaces (not duplicates) — review-time behavior documented in `_review.md`.
- [x] A-027 Scenario: Block present, raw over threshold (`true_impact.net: 150`) — `/fab-status` renders the impact line in yellow.
- [x] A-028 Scenario: Refactor at threshold (`excluding.net: 50`, equal not greater) — soft warning NOT shown.

### Edge Cases & Error Handling

- [x] A-029 `fab impact` exits non-zero with a stderr message when `git merge-base` cannot be resolved; the apply-finish hook skips the `true_impact` write (no partial block) and the stage transition still completes.
- [x] A-030 `_review.md`'s skip-list match (docs/chore/ci) silently skips both passes — no findings emitted, no errors raised, no `## Deletion Candidates` section appended.
- [x] A-031 Existing `.status.yaml` files without `true_impact` remain valid; load/save round-trip preserves the absent block.

### Code Quality

- [x] A-032 Pattern consistency: new Go code in `src/go/fab/internal/impact/` follows the existing internal-package conventions (cobra subcommand registration, yaml.v3 parsing, table-driven tests, `internal/...` import path).
- [x] A-033 No unnecessary duplication: `git-pr.md` consumes `fab impact` rather than reimplementing shortstat parsing; `apply-finish` and `hydrate-finish` both reuse the same `WriteTrueImpact` helper.
- [x] A-034 Documentation accuracy (per `config.checklist.extra_categories`): `SPEC-_review.md`, `SPEC-fab-status.md`, `SPEC-git-pr.md` accurately describe the new skill behaviors; `_cli-fab.md` accurately describes the new `fab impact` signature.
- [x] A-035 Cross-references (per `config.checklist.extra_categories`): `_review.md` cross-references `code-review.md`'s `## Parsimony Pass` toggle; `fab-status.md` cross-references the `true_impact` schema; `git-pr.md` cross-references `fab impact`.

## Notes

- Magic numbers are hard-coded per spec assumptions #8, #9, #11: parsimony threshold `100`, refactor warning threshold `+50`, skip list `[docs, chore, ci]`. These MUST NOT become config knobs.
- The only project-level knob is `## Parsimony Pass` `Enabled: true|false` in `code-review.md`.
- The `true_impact` block is created lazily on first apply-finish (no template placeholder per assumption #15).
- The `## Deletion Candidates` section is review-generated, not template-scaffolded (per assumption #18).

## Deletion Candidates

None — this change adds new functionality (true_impact computation, parsimony pass, deletion-candidate prompt, `fab impact` subcommand) without making existing code redundant. The pre-extraction inline shortstat math in `src/kit/skills/git-pr.md` (the only candidate for removal) was already fully removed in the apply diff (replaced by the `fab impact` invocation) — verified by `grep` returning no `git diff --shortstat` references in `git-pr.md`. No further deletion opportunities discovered.
