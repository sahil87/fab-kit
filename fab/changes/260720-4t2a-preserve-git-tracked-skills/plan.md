# Plan: Preserve Git-Tracked Skills

**Change**: 260720-4t2a-preserve-git-tracked-skills
**Intake**: `intake.md`

## Requirements

### Sync: Stale-Skill Cleanup

#### R1: Preserve Git-Tracked Skill Entries
`cleanStaleSkills` SHALL NOT delete a skill-directory entry (directory-format) or skill file
(flat-format) that is tracked by git in the consuming repo, even when the entry's name is absent
from fab-kit's own canonical skill list.

- **GIVEN** a consuming repo has committed a custom skill under `.claude/skills/<name>/SKILL.md`
  (or the equivalent path for `.opencode/commands/`, `.agents/skills/`, `.gemini/skills/`)
- **WHEN** `fab sync` runs and `cleanStaleSkills` evaluates that entry (its name is not in
  fab-kit's own canonical skill list)
- **THEN** the entry is left in place, not deleted
- **AND** a `git ls-files --error-unmatch` check against the entry's repo-relative path is what
  distinguishes "project-committed" from "genuinely stale"

#### R2: Preserve Existing Cleanup for Untracked Entries
`cleanStaleSkills` SHALL continue to delete any skill-directory entry or skill file that is
**not** tracked by git, exactly as before this change.

- **GIVEN** a `.claude/skills/<name>/` directory (or flat file) is a leftover from a retired
  fab-kit skill and was never committed to the consuming repo's own git history
- **WHEN** `fab sync` runs and the entry's name is absent from fab-kit's canonical skill list
- **THEN** the entry is deleted, same as prior behavior

#### R3: Fail Open When Git Is Unavailable
The git-tracked check SHALL fail open (report "not tracked") when git is absent from `PATH` or
`repoRoot` is not inside a git work tree, so behavior in a non-git context is unchanged from
before this fix.

- **GIVEN** `repoRoot` is not a git repository, or the `git` binary cannot be found
- **WHEN** `cleanStaleSkills` evaluates a candidate-for-removal entry
- **THEN** the entry is treated as untracked and deleted if its name isn't in the canonical list
  — identical to the pre-fix behavior

### Design Decisions

#### Git-Tracking Check Over an Explicit Config Allowlist
**Decision**: Determine "project-committed, don't delete" via `git ls-files --error-unmatch`
against the consuming repo, rather than adding a new `config.yaml` allowlist key (e.g.
`skills.keep: [...]`).
**Why**: Zero new config surface, zero migration, and it's self-maintaining — a consumer that
commits a new custom skill is automatically protected without editing any fab-kit-owned config
file. Matches an existing pattern already in this package (`warnIfFabVersionIgnored`'s
`git check-ignore -q` fail-open shape in `init.go`).
**Rejected**: A `config.yaml` allowlist would require every consumer to remember to register each
custom skill by name, re-introducing exactly the kind of manual bookkeeping this fix is meant to
eliminate, and it's a new schema key needing a migration for existing projects.
*Introduced by*: 260720-4t2a-preserve-git-tracked-skills

#### Keep the Fix General Rather Than Redirecting to `.claude/commands/`
**Decision**: Fix `cleanStaleSkills` itself rather than recommending consumers move custom skills
to the untouched `.claude/commands/` directory.
**Why**: `.claude/commands/` has no equivalent of Claude Code's model-driven skill
auto-invocation (it only scans `.claude/skills/*/SKILL.md`) — any skill that wants auto-invocation
has no escape hatch there. Fixing the cleanup logic covers that case; a per-skill redirect
doesn't.
**Rejected**: Redirecting `archive-dormant-repos` (or any similarly explicit-invocation-only
skill) to `.claude/commands/` would work for that one skill but leaves the general bug — and every
other consumer's auto-invoked custom skill — unaddressed.
*Introduced by*: 260720-4t2a-preserve-git-tracked-skills

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add `isGitTracked(repoRoot, relPath string) bool` helper to `src/go/fab-kit/internal/skills.go` — shells out to `git ls-files --error-unmatch --  <relPath>` with `cmd.Dir = repoRoot`, returns `cmd.Run() == nil` <!-- R1 -->
- [x] T002 Wire `isGitTracked` into `cleanStaleSkills`'s directory-format branch: before `os.RemoveAll`, compute `rel, err := filepath.Rel(repoRoot, filepath.Join(baseDir, e.Name()))` and skip removal when `err == nil && isGitTracked(repoRoot, rel)` <!-- R1 -->
- [x] T003 Wire the same check into `cleanStaleSkills`'s flat-format branch, before `os.Remove` <!-- R1 -->

### Phase 3: Integration & Edge Cases

- [x] T004 Confirm untracked entries are still removed in both formats — no behavior change for the pre-existing cleanup path <!-- R2 -->
- [x] T005 Confirm fail-open behavior in a non-git `t.TempDir()` (no `git init`) — existing `TestCleanStaleSkills_Directory`/`_Flat` tests continue to pass unmodified, proving the fail-open path preserves prior behavior <!-- R3 -->

### Phase 4: Polish

- [x] T006 Add `TestCleanStaleSkills_Directory_PreservesGitTrackedCustomSkill` and `TestCleanStaleSkills_Flat_PreservesGitTrackedCustomFile` to `src/go/fab-kit/internal/skills_test.go`, matching the existing file's test-helper conventions (`requireGit`, `t.TempDir()`, `exec.Command("git", "init", ...)`) <!-- R1 -->

## Execution Order

- T001 blocks T002 and T003 (both call the new helper)
- T004/T005 are verification of behavior already exercised by existing tests, not new code — run after T001-T003
- T006 (new tests) written test-first per repo convention: verified red (tests fail against the pre-fix code, reverted via `git stash`) before green (tests pass with the fix restored)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `TestCleanStaleSkills_Directory_PreservesGitTrackedCustomSkill` passes — a git-tracked custom skill directory survives `cleanStaleSkills`
- [x] A-002 R1: `TestCleanStaleSkills_Flat_PreservesGitTrackedCustomFile` passes — a git-tracked custom flat file survives `cleanStaleSkills`

### Behavioral Correctness

- [x] A-003 R2: Both new tests also assert an *untracked* stale entry in the same directory is still removed in the same `cleanStaleSkills` call — proving the fix is additive, not a blanket skip

### Removal Verification

- [x] A-004 R#: N/A — no requirement removed by this change

### Scenario Coverage

- [x] A-005 R3: `TestCleanStaleSkills_Directory`/`_Flat` (pre-existing, non-git `t.TempDir()`) pass unmodified, proving fail-open parity with prior behavior
- [x] A-006 R1: End-to-end with the real compiled `fab-kit` binary (not just `go test`) — built from this fix, ran `fab-kit sync` twice against a repo shaped like the real-world reproduction (git-tracked custom skill + one canonical stock skill): run 1 preserved the custom skill and deployed the canonical one; run 2 (with an added untracked leftover) removed the untracked entry while still preserving the tracked one

### Edge Cases & Error Handling

- [x] A-007 R3: Verified the fail-open path directly — `isGitTracked` returns `false` (not an error) when `repoRoot` has no `.git`, so a candidate entry falls through to the pre-fix delete-if-not-canonical logic exactly as before

### Code Quality

- [x] A-008 Pattern consistency: `isGitTracked` mirrors the existing `warnIfFabVersionIgnored` fail-open `exec.Command("git", ...)` shape already in this package (`init.go`) — no new pattern introduced
- [x] A-009 No unnecessary duplication: reuses `filepath.Rel`/`exec.Command` already imported in the package; no new dependency. Review (fresh sub-agent, per this repo's Apply ⇄ Review Loop) flagged the directory- and flat-format branches' git-tracked check as duplicated (nice-to-have) — extracted into a shared `isPreservedByGit(baseDir, entryName, repoRoot string) bool` helper, used by both branches
- [x] A-010 God functions: review flagged this item's original "well under 50 lines" claim as inaccurate (`cleanStaleSkills` was 53 lines) — corrected after the A-009 extraction: the function is now 51 lines total (49-line body excluding signature/closing brace), and remains two cohesive parallel format branches rather than one sprawling body
- [x] A-011 Magic strings: no new string/numeric literals beyond the `git`/`ls-files`/`--error-unmatch` command tokens, which are self-documenting

## Notes

- No `.claude/skills/` deployment concern (Anti-Pattern: "Editing `.claude/skills/` directly") — this change touches `internal/skills.go`, the Go source that *implements* the sync/cleanup mechanics, not a deployed skill copy.
- No SPEC mirror required (Anti-Pattern: "Shipping a skill change without its SPEC mirror") — that constraint applies to `src/kit/skills/*.md` prompt files; this change is internal Go logic with no corresponding `docs/specs/skills/SPEC-*.md`.
- No CLI signature change, so no `_cli-fab.md` update required.

## Deletion Candidates

- None — this change adds new preservation logic; nothing pre-existing becomes redundant or unused.

## Assumptions

(none — the intake's four assumptions covered the full design; apply required no further inline
decisions since the implementation was already written and tested prior to this pipeline run,
and matched the intake's "What Changes" section exactly)

0 assumptions.
