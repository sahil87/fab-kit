# Plan: Per-Change Base Branch — Record `base_branch` in `.status.yaml` and Read It on Every Base-Relative Surface

**Change**: 260915-vy27-per-change-base-branch
**Intake**: `intake.md`

## Requirements

### Schema: `.status.yaml` `base_branch` field

#### R1: Optional `base_branch` scalar with drop-when-empty round-trip
`.status.yaml` MUST accept a top-level optional string `base_branch` holding a **plain branch name** (e.g. `main`, `260914-abcd-parent`), never a remote-tracking ref. The Go `statusfile.StatusFile` SHALL carry `BaseBranch string \`yaml:"base_branch,omitempty"\`` modeled exactly on `summary`: decoded by `Load()`'s explicit key switch, dropped on write when empty, and inserted before `last_updated` on a sparse document that lacks the key. The status template SHALL carry only a comment line (no placeholder), in the style of the existing `# true_impact:` comment. `fab status refresh` MUST NOT write or clear the field.

- **GIVEN** a `.status.yaml` written before this change (no `base_branch` key)
- **WHEN** it is loaded, mutated elsewhere, and saved
- **THEN** no `base_branch:` key appears and every other key round-trips byte-identically
- **AND** when `BaseBranch` is set to `main` before save, `base_branch: main` is inserted before `last_updated`, and reloading yields `main`

### CLI: `fab status set-base-branch` / `get-base-branch`

#### R2: Verb pair modeled on `set-summary` / `get-summary`
`fab status set-base-branch <change> <branch>` MUST write the field under the status flock (`withStatusLock` → `status.SetBaseBranch`), requiring a non-empty branch and NOT validating remote existence. `fab status get-base-branch <change> [--json]` MUST print the value (an empty line when absent) and, with `--json`, emit `{"base_branch":"…"}` (absent → `{"base_branch":""}`), joining the read-only `--json` query surface (nine → ten subcommands).

- **GIVEN** a change with no `base_branch`
- **WHEN** `fab status get-base-branch <change>` runs
- **THEN** it exits 0 and prints an empty line; `--json` prints `{"base_branch":""}`
- **GIVEN** `fab status set-base-branch <change> feature-x` has run
- **WHEN** `get-base-branch` runs
- **THEN** it prints `feature-x`; `--json` prints `{"base_branch":"feature-x"}`
- **GIVEN** `set-base-branch <change> ""`
- **THEN** the command exits non-zero with an actionable message and writes nothing

### Go: one shared base-resolution helper

#### R3: `impact.ResolveBaseRef` and `impact.MergeBase` replace the two private helpers
`src/go/fab/internal/impact` SHALL export `ResolveBaseRef(repoDir, baseBranch string) string` returning the remote-tracking ref to diff against in this order: `origin/<baseBranch>` when `baseBranch` is non-empty AND `git rev-parse --verify refs/remotes/origin/<baseBranch>` succeeds; else the target of `origin/HEAD` (`git symbolic-ref --short refs/remotes/origin/HEAD`); else `origin/main`; else `origin/master`; else `""`. It SHALL export `MergeBase(repoDir, baseRef string) (string, error)` returning the trimmed `git merge-base <baseRef> HEAD`, erroring on an empty ref or a failed merge-base. Both run git pinned to `repoDir` (empty ⇒ process cwd). `prmeta.mergeBase` and `status.resolveMergeBase` MUST be deleted, not wrapped.

- **GIVEN** a repo whose `origin/HEAD` → `origin/develop` and no `origin/main`/`origin/master`
- **WHEN** `ResolveBaseRef(repo, "")` runs
- **THEN** it returns `origin/develop` (today: nothing resolves)
- **GIVEN** `base_branch: parent` and `origin/parent` exists
- **WHEN** `ResolveBaseRef(repo, "parent")` runs
- **THEN** it returns `origin/parent`
- **GIVEN** `base_branch: parent` but `origin/parent` was deleted
- **THEN** it falls through to the default chain (fail-open)
- **GIVEN** no remote refs at all
- **THEN** it returns `""` and `MergeBase(repo, "")` returns an error

#### R4: `fab pr-meta` and `WriteTrueImpact` pass the change's `base_branch` to the helper
`prmeta.Gather` and `status.WriteTrueImpact` MUST resolve the diff base via `impact.ResolveBaseRef(repoDir, statusFile.BaseBranch)` + `impact.MergeBase`, keeping their existing degradation contracts (pr-meta drops only the Impact block; `WriteTrueImpact` warns on stderr and returns nil). `fab impact <base> <head>` and `fab change list --show-stats` are unchanged.

- **GIVEN** branch `child` created off `parent` (which has its own commits past `main`) with `base_branch: parent`
- **WHEN** `fab pr-meta` renders Impact and `fab status finish <change> ship` writes `true_impact`
- **THEN** both count only `child`'s own commits, excluding `parent`'s
- **GIVEN** the same change with `base_branch` absent
- **THEN** both measure against the resolved default branch exactly as before

### Skills: who writes `base_branch`

#### R5: Branch creation records the base (`/git-branch` Step 4, `/fab-new` Step 11)
Both branch-creation twins MUST record `base_branch` via `fab status set-base-branch "{name}" "$base_branch"` after every create/rename/`--track` action, and write-if-absent (probe `get-base-branch` first) on the already-active / checked-out no-op cases. `/git-branch` SHALL gain an optional `--base <branch>` argument that overrides the default; when absent, `base_branch` is the g8st default-branch chain (`origin/HEAD` → `gh repo view` → main/master probe), inlined in each twin. `/fab-new` Step 11 takes no `--base` (it always uses the chain). `git-branch.md` § Key Properties "Modifies `.status.yaml`?" becomes **Yes** with the reason. The two `<!-- Keep these cases in sync -->` comments stay accurate. A standalone-branch fallback (explicit name matching no change) writes nothing (no `.status.yaml` exists).

- **GIVEN** `/git-branch` on `main` for change X with no `--base`
- **WHEN** the branch is created
- **THEN** `.status.yaml` gains `base_branch: main` and the report line is unchanged
- **GIVEN** `/git-branch --base parent` for change X
- **THEN** `base_branch: parent` is recorded regardless of the current branch
- **GIVEN** `/git-branch` re-run on an already-active branch whose `.status.yaml` already has `base_branch: parent`
- **THEN** the value is untouched (idempotent, Constitution III)

#### R6: Operator `stacked-prs` mode records the dependency branch and drops the post-create retarget
In `fab-operator.md` § Same-repo resolution (`stacked-prs` mode), after the dependent's branch is created off the dependency's branch, the operator MUST run `fab status set-base-branch <change> <dep-branch>` in the target worktree. The sentence instructing a post-create `gh pr edit <pr> --base <dep-branch>` retarget ("`/git-pr` itself is unchanged and mode-unaware") MUST be removed. The Ordered Merge retarget-verify / `rebase --onto` steps and the "Why `origin/{default_branch}` as base" paragraph are unchanged.

- **GIVEN** a `stacked-prs` queue with B depending on A
- **WHEN** B is spawned
- **THEN** B's `.status.yaml` has `base_branch: <A-branch>` before `/fab-fff` runs, and `/git-pr` creates B's PR against `<A-branch>` with no operator retarget

### Skills: who reads `base_branch`

#### R7: Review dispatch, `/fab-adopt`, and `/git-pr` resolve the base with one shared snippet
Each consumer MUST resolve the base as: `fab status get-base-branch {id}` → the g8st chain when empty → verify `refs/remotes/origin/$base_branch` exists, else fall back to the chain (vanished stacked base) → `git merge-base HEAD origin/$base_branch`. Specifically: `_review.md` § Review Agent Dispatch replaces the `origin/main` merge-base sentence with the snippet (diff and `--name-only` list share `<base>`); `fab-adopt.md` Step 0 uses the adopted PR's `gh pr view --json baseRefName` when a PR exists (else the chain) and Step 1 records it with `set-base-branch` after `fab change new`, and `_generation.md`'s Intake-from-Diff "default-branch merge-base" wording becomes "base-branch merge-base"; `git-pr.md` derives `base_branch` in Step 1 next to `default_branch` and passes `--base "$base_branch"` on both `gh pr create` paths (normal and `--fill`), while the Step 2 default-branch guard keeps using `default_branch`.

- **GIVEN** a stacked change with `base_branch: parent`
- **WHEN** the review worker computes its diff
- **THEN** it diffs `merge-base(HEAD, origin/parent)...HEAD` and never sees `parent`'s commits
- **GIVEN** `/git-pr` on that change
- **THEN** `gh pr create --draft --base parent …` is issued
- **GIVEN** `/fab-adopt` on a branch whose OPEN PR has `baseRefName: parent`
- **THEN** the adopt diff base is `origin/parent` and the new change records `base_branch: parent`

### Reference, migration, sweep

#### R8: CLI reference updated for every changed command
`_cli-fab.md` MUST gain the `set-base-branch` / `get-base-branch` row in § fab status, count ten read-only `--json` query subcommands, rewrite § fab pr-meta's "Impact math" bullet to the new resolution order, and name the shared helper's order in § fab impact "Consumers".

- **GIVEN** the CLI doc test suite (`clifab_doc_test.go`)
- **WHEN** it runs
- **THEN** it passes with both verbs documented

#### R9: Announce migration for the schema addition
`src/kit/migrations/2.27.0-to-2.28.0.md` MUST exist with Summary (new optional field + verbs, fail-open), Pre-check (`fab status get-base-branch <any-change>` prints an empty line), **Changes: None**, and Verification (`validate-status-file` exits 0 on an untouched change; `set-base-branch` → `get-base-branch` round-trips). The exact version range is release-owned.

- **GIVEN** a project on 2.27.0 upgrading
- **WHEN** `/fab-setup migrations` runs
- **THEN** the migration is listed, edits nothing, and verification passes

#### R10: Sibling sweep and Constitution V
Every stale claim in the sweep class MUST be updated up front: `docs/specs/glossary.md` `/git-branch` row ("never modifies fab state"), `docs/specs/skills.md` `/git-branch` and `/fab-adopt` Step 0 wording (default-branch merge-base), `docs/specs/templates.md` `.status.yaml` field notes (add `base_branch`). No `src/kit/**` edit may cite `src/go/*` or fab-kit's `docs/specs/*`, `docs/memory/*`, `docs/site/*`; the kit portability test MUST pass.

- **GIVEN** `grep -rn 'origin/main\|mode-unaware\|never modifies fab state' src/kit/skills docs/specs`
- **WHEN** apply finishes
- **THEN** the only remaining `origin/main` hits are the literal-fallback probe lines of the g8st chain and the operator's cherry-pick base prose

### Non-Goals

- Dependency-branch drift after a dependent PR exists — out of scope, as today.
- Changing `/git-pr-review` or the operator's Dependency Resolution step 0 — both already use the real base.
- Rewriting historical `true_impact` blocks on archived or in-flight changes.
- An explicit `--base` flag on `fab pr-meta` / `fab status finish` — the recorded field is the sole input; `set-base-branch` is the override.
- A base column in `fab change list --show-stats` — it inherits corrected numbers only.

### Design Decisions

#### `base_branch` stores a plain branch name
**Decision**: `base_branch` holds `main` / `<branch>`, never `origin/<branch>`; consumers prefix `origin/` when they need the remote-tracking ref.
**Why**: `gh pr create --base` / `gh pr edit --base` take plain names, every skill's `default_branch` variable already holds one, and a plain name survives a remote rename.
**Rejected**: storing the remote-tracking ref — saves one string concatenation in Go but leaks the remote name into the schema and mismatches every `gh` call site.
*Introduced by*: 260915-vy27-per-change-base-branch

#### The status field is the only input; no per-command `--base` override
**Decision**: `fab pr-meta` and `fab status finish` read `base_branch` from `.status.yaml` and take no `--base` flag.
**Why**: a flag re-creates the per-caller-must-remember drift the change exists to remove; `fab status set-base-branch` then re-run is the override path and leaves a record.
**Rejected**: `--base` on both commands — two more flags, two more doc rows, and a second source of truth that can disagree with the file.
*Introduced by*: 260915-vy27-per-change-base-branch

#### `/git-branch --base <branch>` is the human path; the verb is the operator path
**Decision**: `/git-branch` gains an optional `--base <branch>`; `/fab-new` Step 11 does not (it always records the resolved default). The operator writes the field directly with `fab status set-base-branch` after its own branch step.
**Why**: a human building a stack by hand needs a way to say "this is off `parent`" at branch time; the operator already knows the dependency branch and needs no skill argument.
**Rejected**: verb-only (humans would have to remember a second command after branching); `--base` on `/fab-new` too (its interactive intake moment is the wrong place for a git-topology argument).
*Introduced by*: 260915-vy27-per-change-base-branch

#### Shared helper lives in `internal/impact`
**Decision**: `ResolveBaseRef` / `MergeBase` are exported from `internal/impact`.
**Why**: both consumers (`prmeta`, `status`) already import it for the diff math, so no new package and no import-cycle risk; the merge-base is the first step of the same computation.
**Rejected**: a new `internal/gitbase` package — cleaner name, one more package for two functions.
*Introduced by*: 260915-vy27-per-change-base-branch

#### Fail-open on a vanished base
**Decision**: when `base_branch` is set but `origin/<base_branch>` no longer resolves, Go and skill consumers fall through to the default-branch chain instead of erroring.
**Why**: the typical cause is the dependency PR merging and its branch being deleted — at that point the default branch *is* the correct base; erroring would break `fab status finish` and `pr-meta` on every completed stack.
**Rejected**: hard error — surfaces the stale field but blocks the pipeline for a condition that has a correct answer.
*Introduced by*: 260915-vy27-per-change-base-branch

## Tasks

### Phase 1: Setup — Go core

- [x] T001 `src/go/fab/internal/statusfile/statusfile.go`: add `BaseBranch string \`yaml:"base_branch,omitempty"\`` after `ChangeTypeSource`; add `case "base_branch"` to `Load()`'s key switch; add the `syncToRaw` `case "base_branch"` (drop when empty, else set value) and the `!seen["base_branch"] && sf.BaseBranch != ""` `insertKey` line next to `summary`'s. Tests in `statusfile_test.go`: `TestBaseBranch_AbsentStaysAbsent`, `TestBaseBranch_RoundTrips`, `TestBaseBranch_InsertedIntoSparseDoc` (modeled on the `summary` trio); update `golden_test.go` only if its fixture enumerates keys. <!-- R1 -->
- [x] T002 [P] `src/go/fab/internal/status/status.go`: add `SetBaseBranch(statusFile, statusPath, branch string) error` next to `SetSummary` — returns an error on empty `branch`, otherwise sets and `Save`s. Test in `mutators_test.go`. <!-- R2 -->
- [x] T003 `src/go/fab/cmd/fab/status.go`: add `statusSetBaseBranchCmd()` (`Use: "set-base-branch <change> <branch>"`, `ExactArgs(2)`, `withStatusLock` → `status.SetBaseBranch`) and `statusGetBaseBranchCmd()` (`Use: "get-base-branch <change>"`, `--json` → `baseBranchJSON{BaseBranch string \`json:"base_branch"\`}`, plain → `fmt.Println(sf.BaseBranch)`); register both in the subcommand list. Tests: `status_test.go` `Use`-prefix checks, `status_json_test.go` `TestStatusGetBaseBranchJSON` + `_EmptyIsEmptyString`, and add `"get-base-branch"` to `TestStatusQueryCmds_HaveJSONFlag`. <!-- R2 -->
- [x] T004 [P] `src/go/fab/internal/impact/impact.go`: add `ResolveBaseRef(repoDir, baseBranch string) string` and `MergeBase(repoDir, baseRef string) (string, error)` per R3 (git pinned via `cmd.Dir`). Table-driven tests in `impact_test.go` for the six R3 cases using a temp repo with `git update-ref refs/remotes/origin/…` and `git symbolic-ref refs/remotes/origin/HEAD`. <!-- R3 -->

### Phase 2: Core Implementation — Go consumers + template

- [x] T005 `src/go/fab/internal/status/true_impact.go`: delete `resolveMergeBase`; in `WriteTrueImpact` call `impact.ResolveBaseRef(repoDir, statusFile.BaseBranch)` then `impact.MergeBase`, keeping the stderr-warn-and-return-nil contract (message names the resolution order). `true_impact_test.go`: keep existing cases passing; add `TestWriteTrueImpact_OriginHeadDevelop` (only `origin/develop` + `origin/HEAD`, expects a block) and `TestWriteTrueImpact_StackedBaseExcludesParent` (`base_branch: parent`, parent has an extra commit past main; assert counts exclude it). <!-- R4 -->
- [x] T006 [P] `src/go/fab/internal/prmeta/prmeta.go`: delete `mergeBase`; in `Gather` resolve via `impact.ResolveBaseRef(repoDir, status.BaseBranch)` + `impact.MergeBase`, unchanged degradation. `prmeta_test.go`: add a stacked-branch case asserting the Impact table excludes the parent's lines, and an `origin/HEAD`-only case asserting `HasImpact`. <!-- R4 -->
- [x] T007 [P] `src/kit/templates/status.yaml`: add `# base_branch: written by /git-branch or /fab-new when the change's branch is created (no placeholder here).` directly above the existing `# true_impact:` comment. <!-- R1 -->

### Phase 3: Integration — skills

- [x] T008 `src/kit/skills/git-branch.md`: Arguments gains `--base <branch>` (optional; overrides the recorded base); Step 4 gains a **Record the base** sub-step after the action (create / rename / `--track` → always write; already-active / checked-out → write only when `fab status get-base-branch` prints empty; standalone-branch fallback → skip) using the g8st chain inlined + `fab status set-base-branch "{name}" "$base_branch"`; Key Properties row "Modifies `.status.yaml`?" → `Yes — writes base_branch via fab status set-base-branch (create/rename/track always; no-op cases only when absent; --base overrides)`; update the in-sync HTML comment to name the new sub-step. <!-- R5 -->
- [x] T009 `src/kit/skills/fab-new.md` Step 11: mirror T008's sub-step after the case table (no `--base`; chain only; same write-if-absent rule for rows 1–2), update its in-sync comment identically. <!-- R5 -->
- [x] T010 [P] `src/kit/skills/fab-operator.md` § Same-repo resolution (`stacked-prs` mode) (~line 582): after the branch step add `fab status set-base-branch <change> <dep-branch>` (run in the target worktree); delete the "After `/git-pr` creates the dependent's PR, the operator retargets … (`/git-pr` itself is unchanged and mode-unaware)" sentence; adjust the `stacked-prs` bullet (~line 664) "its PR targets the dependency's branch" to say `/git-pr` targets it via the recorded base. Leave Ordered Merge (~716–725) and the "Why `origin/{default_branch}` as base" paragraph untouched. <!-- R6 -->
- [x] T011 [P] `src/kit/skills/_review.md` § Review Agent Dispatch (~line 55): replace the diff/name-only bullets' base derivation with the R7 snippet (`fab status get-base-branch {id}` → chain → existence check → merge-base); note that the sequencer passes `{id}`. <!-- R7 -->
- [x] T012 [P] `src/kit/skills/fab-adopt.md` Step 0 item 4: derive `base_branch` from `gh pr view --json baseRefName -q .baseRefName` when `{pr_state}` is `OPEN`, else `default_branch`; verify `origin/$base_branch` exists (else `default_branch`); `base=$(git merge-base HEAD "origin/$base_branch")`; the empty-diff STOP names `{base_branch}`. Steps 1+2 item 1: after `fab change new`, run `fab status set-base-branch {name} "$base_branch"`. `src/kit/skills/_generation.md` ~line 158: "default-branch merge-base" → "base-branch merge-base". <!-- R7 -->
- [x] T013 [P] `src/kit/skills/git-pr.md`: Step 1 adds `base_branch=$(fab status get-base-branch "{name}" 2>/dev/null)` (only when `{has_fab}`), falling back to `$default_branch`, then the `origin/$base_branch` existence check (else `default_branch`); Step 1 "Determine" list gains **base_branch**; Step 3c item 4 becomes `gh pr create --draft --base "$base_branch" --title … --body …` and the `--fill` fallback becomes `gh pr create --draft --base "$base_branch" --fill`; Step 2 guard text unchanged. <!-- R7 -->
- [x] T014 `src/kit/skills/_cli-fab.md`: § fab status table — add the `set-base-branch` / `get-base-branch` row after `set-summary`/`get-summary` (mirroring its shape: flock write, empty line on absence, `{"base_branch":"…"}`, no stage auto-populates it except the branch-creation skills); `--json` paragraph "nine" → "ten" and add `get-base-branch` to the list; § fab pr-meta "Impact math" bullet → "against the merge-base of HEAD vs the change's recorded `base_branch` (`origin/<base_branch>`, when set and resolvable), else `origin/HEAD`'s target, else `origin/main`, else `origin/master`"; § fab impact "Consumers" paragraph names the same order for `fab pr-meta` and the finish hooks. <!-- R8 -->

### Phase 4: Polish — migration, sweep, verification

- [x] T015 [P] Create `src/kit/migrations/2.27.0-to-2.28.0.md` per R9 (announce shape of `2.23.17-to-2.24.0` / `2.25.1-to-2.26.0`: Summary, Pre-check, `## Changes` = None, Verification; state the version slot is release-owned). <!-- R9 -->
- [x] T016 [P] Spec sweep: `docs/specs/glossary.md` `/git-branch` row → "writes `base_branch` to the change's `.status.yaml` at branch creation; otherwise never modifies fab state"; `docs/specs/skills.md` `/git-branch` section (~956+) adds the `--base` argument + record-the-base step and its Key Properties; `/fab-adopt` lines ~685/690 "default-branch merge-base" → "base-branch merge-base (PR `baseRefName` when a PR exists)"; `docs/specs/templates.md` `.status.yaml` notes (~58–74) add a `base_branch` bullet after `summary`. Then `grep -rn 'origin/main\|mode-unaware\|never modifies fab state\|default-branch merge-base' src/kit docs/specs` and fix any remaining stale claim outside the g8st literal-fallback probe lines. <!-- R10 -->
- [x] T017 Verify: `gofmt -l src/go` prints nothing; `go test ./src/go/fab/internal/statusfile/... ./src/go/fab/internal/status/... ./src/go/fab/internal/prmeta/... ./src/go/fab/internal/impact/... ./src/go/fab/cmd/fab/...` passes; `go test ./src/go/fab-kit/cmd/fab/...` (kit portability + CLI doc tests) passes; `go build ./src/go/...` succeeds. Fix any failure at its root (Constitution VII). <!-- R3 -->

## Execution Order

- T001 blocks T002, T003, T005, T006 (they read `BaseBranch`)
- T004 blocks T005, T006
- T008 blocks T009 (twin mirrors T008's exact sub-step text)
- T017 runs last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `StatusFile.BaseBranch` exists with `omitempty`, is decoded by `Load`, dropped when empty on save, and inserted before `last_updated` on a sparse document; the status template carries the comment line and no placeholder
- [x] A-002 R2: `fab status set-base-branch <change> <branch>` and `fab status get-base-branch <change> [--json]` exist, are registered, and behave per R2 (empty line / `{"base_branch":""}` when absent)
- [x] A-003 R3: `impact.ResolveBaseRef` and `impact.MergeBase` exist with the documented order; `prmeta.mergeBase` and `status.resolveMergeBase` are gone (grep finds neither)
- [x] A-004 R4: `prmeta.Gather` and `status.WriteTrueImpact` call the shared helper with `statusFile.BaseBranch`
- [x] A-005 R5: `git-branch.md` documents `--base <branch>`, the record-the-base sub-step for every Step 4 case, and the updated Key Properties row; `fab-new.md` Step 11 mirrors the sub-step (chain only)
- [x] A-006 R6: `fab-operator.md` stacked-prs paragraph runs `fab status set-base-branch <change> <dep-branch>` and no longer instructs a post-create `gh pr edit --base` retarget; Ordered Merge steps are unchanged
- [x] A-007 R7: `_review.md`, `fab-adopt.md`, and `git-pr.md` resolve the base with the shared snippet; `git-pr.md` passes `--base` on both `gh pr create` paths; `_generation.md` says "base-branch merge-base"
- [x] A-008 R8: `_cli-fab.md` § fab status has the new row, counts ten `--json` subcommands, and § fab pr-meta / § fab impact state the new resolution order
- [x] A-009 R9: `src/kit/migrations/2.27.0-to-2.28.0.md` exists with Summary / Pre-check / Changes: None / Verification
- [x] A-010 R10: glossary, skills, and templates specs are swept; `grep -rn 'mode-unaware\|never modifies fab state\|default-branch merge-base' src/kit docs/specs` returns nothing — *(review note: the grep returns one line, `docs/specs/glossary.md:78` "otherwise never modifies fab state" — the qualified, accurate wording T016 itself prescribes; no stale unqualified claim remains)*

### Behavioral Correctness

- [x] A-011 R3: with only `origin/develop` and `origin/HEAD → origin/develop`, `ResolveBaseRef(repo, "")` returns `origin/develop` (test exists and passes)
- [x] A-012 R4: with `base_branch: parent` and parent carrying an extra commit past main, `WriteTrueImpact` and `pr-meta` Impact counts exclude the parent's lines (tests exist and pass)
- [x] A-013 R1: a pre-existing `.status.yaml` without `base_branch` round-trips with no new key (test exists and passes)
- [x] A-014 R5: re-running `/git-branch` on an already-active branch leaves an existing `base_branch` value untouched (documented write-if-absent)

### Scenario Coverage

- [x] A-015 R7: for a stacked change, `/git-pr`'s create command line carries `--base "$base_branch"` and `/fab-adopt` prefers the PR's `baseRefName`
- [x] A-016 R6: a `stacked-prs` dependent's PR targets the dependency branch at creation with no operator retarget step

### Edge Cases & Error Handling

- [x] A-017 R3: `base_branch` set but `origin/<base_branch>` missing → falls through to the default chain (test exists and passes); no remote refs at all → `ResolveBaseRef` returns `""` and `WriteTrueImpact` warns and returns nil
- [x] A-018 R2: `set-base-branch` with an empty branch exits non-zero and writes nothing
- [x] A-019 R5: `/git-branch <standalone-name>` (no matching change) writes no `.status.yaml` and does not error on the missing file

### Code Quality

- [x] A-020 Pattern consistency: the verb pair, mutator, JSON struct, and statusfile cases follow the `summary` precedent line for line
- [x] A-021 No unnecessary duplication: exactly one merge-base helper remains in `src/go`; the skill snippet is inlined per consumer (g8st convention) with identical wording
- [x] A-022 Canonical source only: no edits under `.agents/skills/` or `.claude/skills/`
- [x] A-023 CLI ⇒ docs + tests: every new/changed command has a `_cli-fab.md` row and Go tests in the same change
- [x] A-024 Migrations for user-data restructuring: the `.status.yaml` schema addition ships a `src/kit/migrations/` file
- [x] A-025 Deployed content cites no fab-kit-only paths: `go test ./src/go/fab-kit/cmd/fab/...` (kit portability guard) passes
- [x] A-026 Sibling sweep done up front: `git-branch.md` ↔ `fab-new.md` Step 11 twins and the aggregate specs (`glossary.md`, `skills.md`, `templates.md`) are updated in apply, not left to review
- [x] A-027 Test integrity: no implementation was bent to satisfy a fixture; `gofmt -l src/go` is empty

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Memory files (`pipeline/schemas`, `pipeline/change-lifecycle`, `pipeline/execution-skills`, `runtime/operator`, `distribution/migrations`) are hydrate's job, not apply's.

## Deletion Candidates

- None — this change's only removals (`prmeta.mergeBase`, `status.resolveMergeBase`) were planned in R3 and already executed; review found no further code the change makes redundant or unused.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | `base_branch` stores a plain branch name; consumers prefix `origin/` (resolves intake #14) | `gh --base` takes plain names; every skill's `default_branch` variable already holds one | S:70 R:80 A:85 D:80 |
| 2 | Confident | No `--base` flag on `fab pr-meta` / `fab status finish`; the field is the sole input (resolves intake #15) | A flag re-creates per-caller drift; `set-base-branch` then re-run is the override and leaves a record | S:65 R:85 A:80 D:75 |
| 3 | Confident | `/git-branch` gains `--base <branch>`; `/fab-new` Step 11 does not; the operator writes the field via the verb (resolves intake #16) | Humans need a branch-time way to name the base; the operator already knows it; `/fab-new`'s intake moment is the wrong place | S:65 R:80 A:75 D:70 |
| 4 | Confident | `fab change list --show-stats` is unchanged (resolves intake #17) | Corrected `true_impact` numbers flow through; a base column is speculative UI | S:70 R:90 A:85 D:80 |
| 5 | Confident | Shared helper lives in `internal/impact` as `ResolveBaseRef` + `MergeBase` | Both consumers already import it; no new package or cycle risk; placement is trivially reversible | S:60 R:90 A:80 D:70 |
| 6 | Confident | `SetBaseBranch` rejects an empty branch (unlike `SetSummary`, which clears) | An empty base is never meaningful; clearing is not a use case the intake names | S:60 R:85 A:80 D:70 |
| 7 | Confident | Write-if-absent on `/git-branch` no-op cases is probed with `fab status get-base-branch` (empty line ⇒ write) | Reuses the read verb; keeps the skill idempotent without a new `set-if-absent` flag | S:65 R:85 A:80 D:75 |
| 8 | Tentative | Skill snippets verify `refs/remotes/origin/$base_branch` without a `git fetch` first | Fetching in every consumer would slow review dispatch; the operator already fetches at step 0; a stale local ref degrades to the default chain, not an error | S:50 R:75 A:70 D:55 |

8 assumptions (0 certain, 7 confident, 1 tentative).
