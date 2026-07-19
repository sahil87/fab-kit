# Plan: Kit Upgrade 2.16.3 → 2.16.4

**Change**: 260719-6kcj-kit-upgrade-2-16-4
**Intake**: `intake.md`

> Carry-only chore — the fab-kit 2.16.3 → 2.16.4 upgrade diff is ALREADY APPLIED to the working tree (uncommitted). This plan does NOT re-run `fab upgrade-repo` or edit the stamped files; its tasks VERIFY the working-tree diff is exactly the expected 3-file version stamp and keep it intact for shipping.

## Requirements

### Distribution: Kit Version Stamp Carry

#### R1: The working-tree diff MUST be exactly the three expected version-stamp files
The change SHALL carry a working-tree diff limited to exactly three modified files — `fab/.fab-version`, `fab/.kit-migration-version`, and `fab/project/config.yaml` — plus this change's own artifacts under `fab/changes/260719-6kcj-kit-upgrade-2-16-4/`. No other source, kit, or config file may appear in the diff.

- **GIVEN** the fab-kit 2.16.3 → 2.16.4 upgrade was already applied to the working tree by `fab upgrade-repo`
- **WHEN** `git status --porcelain` is inspected
- **THEN** exactly `fab/.fab-version`, `fab/.kit-migration-version`, and `fab/project/config.yaml` show as modified (`M`)
- **AND** the only untracked entry is the change's own folder `fab/changes/260719-6kcj-kit-upgrade-2-16-4/`

#### R2: The two version-stamp files MUST bump 2.16.3 → 2.16.4 and nothing else
Both `fab/.fab-version` and `fab/.kit-migration-version` SHALL contain exactly `2.16.4` after the upgrade, changed from `2.16.3`, with no other content difference.

- **GIVEN** the applied upgrade diff
- **WHEN** `git diff fab/.fab-version` and `git diff fab/.kit-migration-version` are inspected
- **THEN** each shows a single-line change `-2.16.3` / `+2.16.4`
- **AND** the current file contents are exactly `2.16.4`

#### R3: The config.yaml change MUST be limited to the reference-fence header comment
The `fab/project/config.yaml` diff SHALL be limited to the single reference-fence header comment line changing `kit 2.16.3` → `kit 2.16.4`. No configuration field value, above-fence override, or other comment may change.

- **GIVEN** the applied upgrade diff
- **WHEN** `git diff fab/project/config.yaml` is inspected
- **THEN** the only changed line is `# >>> fab reference (kit 2.16.3) >>>` → `# >>> fab reference (kit 2.16.4) >>>`
- **AND** no field values (`agent`, `checklist`, `project`, `providers`, `source_paths`, `test_paths`, `true_impact_exclude`) and no above-fence overrides differ

#### R4: The stamped files MUST be kept intact — no revert, regenerate, or amend
No task in this change SHALL re-run `fab upgrade-repo`, edit, revert, regenerate, or amend the three stamped files. The only files this change may create or modify are its own pipeline artifacts under `fab/changes/260719-6kcj-kit-upgrade-2-16-4/`.

- **GIVEN** the verified upgrade diff
- **WHEN** apply executes its tasks
- **THEN** the three stamped files remain byte-identical to their post-upgrade state
- **AND** the working tree still shows exactly the R1 file set at apply completion

### Non-Goals

- Producing or re-running the upgrade — the diff is already applied; this change only carries it.
- Any `src/` change (Go or kit content) — no behavior change in this repo's sources.
- Any `docs/memory/` update — the upgrade mechanism is unchanged; a mechanical version stamp changes no spec-level behavior (intake § Affected Memory: none).
- Test runs — no `.go` files are touched, so no test scope applies (code-quality test strategy keys test scope on changed `.go` files).

### Design Decisions

#### Carry the pre-applied upgrade diff through the pipeline rather than committing directly
**Decision**: Shepherd the already-applied 3-file version-stamp diff through the full fab pipeline as a PR against `main`, rather than committing it directly to `main`.
**Why**: Traceability — a change record plus a reviewable PR, departing from the direct-to-main precedent of `521719c5` "Upgrading fab-kit" (the prior 2.16.2 → 2.16.3 bump of identical shape).
**Rejected**: Direct commit to `main` (the `521719c5` precedent) — faster but leaves no change record or PR trail for the bump.
*Introduced by*: 260719-6kcj-kit-upgrade-2-16-4

## Tasks

### Phase 1: Verify diff exactness

- [x] T001 Verify `git status --porcelain` shows exactly the three modified files (`fab/.fab-version`, `fab/.kit-migration-version`, `fab/project/config.yaml`) plus only the untracked `fab/changes/260719-6kcj-kit-upgrade-2-16-4/` folder — no other modified/untracked entries <!-- R1 -->
- [x] T002 Verify `git diff fab/.fab-version` and `git diff fab/.kit-migration-version` each show a single-line `-2.16.3`/`+2.16.4` change and the files' current contents are exactly `2.16.4` <!-- R2 -->
- [x] T003 Verify `git diff fab/project/config.yaml` changes only the reference-fence header comment (`kit 2.16.3` → `kit 2.16.4`), with no field-value or above-fence-override differences <!-- R3 -->

### Phase 2: Keep intact

- [x] T004 Confirm no task edited, reverted, regenerated, or amended the three stamped files, and that no `fab upgrade-repo` was re-run — the only files created this change are its own artifacts under `fab/changes/260719-6kcj-kit-upgrade-2-16-4/`; re-confirm the working tree still shows exactly the R1 file set <!-- R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `git status --porcelain` shows exactly `fab/.fab-version`, `fab/.kit-migration-version`, and `fab/project/config.yaml` modified, with the change's own folder as the only untracked entry
- [x] A-002 R2: both `fab/.fab-version` and `fab/.kit-migration-version` bump `2.16.3` → `2.16.4` (single-line diff each) and now read exactly `2.16.4`
- [x] A-003 R3: `fab/project/config.yaml` differs only by the reference-fence header comment `kit 2.16.3` → `kit 2.16.4`, with no field-value or above-fence-override change

### Behavioral Correctness

- [x] A-004 R4: the three stamped files are byte-identical to their post-upgrade state — nothing in this change reverted, regenerated, or amended them, and `fab upgrade-repo` was not re-run

### Scenario Coverage

- [x] A-005 R1: at apply completion the working tree still shows exactly the R1 file set (three stamped files modified + the change folder untracked)

### Code Quality

- [x] A-006 Canonical source only: no edit was made under `.claude/skills/` (gitignored deployed copies); no ad-hoc script or subcommand was introduced (code-review.md project rules)
- [x] A-007 Markdown-only artifacts: the change's own artifacts are plain markdown/YAML in standard CommonMark (Constitution IV)

### Documentation Accuracy

- [x] A-008: no memory or spec update is warranted — the upgrade mechanism (documented in the `distribution` domain) is unchanged and a mechanical version stamp changes no spec-level behavior (intake § Affected Memory: none)

### Cross References

- [x] A-009: the change introduces no new claims requiring sibling/mirror sweeps — no `src/kit/skills/*.md`, `SPEC-*.md`, `_cli-fab.md`, or `.go` file is touched (code-quality.md § Sibling & Mirror Sweeps)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- This is a verification-only chore: no source is written, no tests run. Tasks assert the pre-applied diff's exactness and integrity.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Requirements are verification requirements (diff exactness + integrity), not implementation requirements — apply verifies and keeps intact, it produces no code | Intake § What Changes is explicit: "Verify, don't produce"; carry-only chore, no `src/` change | S:95 R:90 A:95 D:95 |
| 2 | Certain | Change type is `chore`; no test scope applies (no `.go` files touched) | Intake § Impact states change type chore and no tests affected; code-quality test strategy keys scope on changed `.go` files | S:95 R:90 A:95 D:95 |
| 3 | Certain | Affected Memory: none — no hydrate content beyond index regen | Intake § Affected Memory: none; upgrade mechanism unchanged, mechanical stamp changes no spec-level behavior | S:90 R:90 A:95 D:90 |

3 assumptions (3 certain, 0 confident, 0 tentative).
