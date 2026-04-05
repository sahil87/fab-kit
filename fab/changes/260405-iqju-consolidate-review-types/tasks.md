# Tasks: Consolidate Review Types

**Change**: 260405-iqju-consolidate-review-types
**Spec**: `spec.md`
**Intake**: `intake.md`

<!--
  TASK FORMAT: - [ ] {ID} [{markers}] {Description with file paths}

  Markers (optional, combine as needed):
    [P]   — Parallelizable (different files, no dependencies on other [P] tasks in same group)

  IDs are sequential: T001, T002, ...
  Include exact file paths in descriptions.
  Each task should be completable in one focused session.

  Tasks are grouped by phase. Phases execute sequentially.
  Within a phase, [P] tasks can execute in parallel.
-->

## Phase 1: Setup

<!-- No build steps needed — this is a pure markdown/skill change. T001 is a read-and-understand task to ground implementation. -->

- [x] T001 Read and internalize current Review Behavior in `src/kit/skills/fab-continue.md` (§ Review Behavior, all sub-sections) and `src/kit/skills/git-pr-review.md` (Phase 2 — Review Request Cascade, Step 2a, Step 2b) in full, to understand exactly what must be extracted vs. removed vs. simplified before writing any file.

## Phase 2: Core Implementation

<!-- Primary functionality. T002 must come first (creates the shared file that T003–T006 reference). T003–T006 are independent of each other once T002 exists. -->

- [x] T002 Create `src/kit/skills/_review.md` — new internal shared skill file following the `_generation.md` pattern. Frontmatter: `name: _review`, `description: "Shared review dispatch — inward spec/tasks/checklist validation and outward holistic diff review used by fab-continue, fab-ff, and fab-fff."`, `user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`. Body: opening block-quote citing fab-continue/fab-ff/fab-fff as orchestrators. Two sections — **Inward Sub-Agent Dispatch** (extracted verbatim from `fab-continue.md` Review Behavior: Preconditions, Sub-Agent Dispatch, Validation Steps, Structured Review Output, Verdict) and **Outward Sub-Agent Dispatch** (new: dispatched via Agent tool `general-purpose`; receives git diff of changed files + changed file list + standard subagent context; full tool access; Codex→Claude cascade with graceful no-op if both unavailable; returns must-fix/should-fix/nice-to-have findings; always-on, no config flag). Both sub-agents run in parallel; findings merged into single must-fix/should-fix/nice-to-have structure feeding the verdict.

- [x] T003 [P] Update `src/kit/skills/fab-continue.md` — replace the inlined Review Behavior body with a delegation reference following the `_generation.md` pattern. Specifically: (1) in the Stage dispatch table (§ Step 3: SRAD + Generation), add row `| review | **Review Behavior** (\`_review.md\`) |`; (2) replace the entire `## Review Behavior` section content with a single-line delegation: "Follow **Review Behavior** (`_review.md`)." — the Preconditions and sub-agent dispatch details move to `_review.md`. The Verdict subsection (pass/fail state transitions, rework options table) MUST remain in `fab-continue.md` as it is orchestration logic, not review procedure.

- [x] T004 [P] Update `src/kit/skills/git-pr-review.md` — simplify Phase 2 to Copilot-only: (1) Replace the "Phase 2 — Review Request Cascade" block with the new Copilot-only flow: attempt `gh pr edit {number} --add-reviewer copilot`; on success poll `gh pr view --json reviews` every 30s up to 20 attempts (10 minutes); when Copilot review appears proceed to Step 3; if timeout print `Copilot review requested but not yet available. Re-run /git-pr-review to process when ready.` and STOP (clean finish); on non-zero exit print `No automated reviewer available. Run /git-pr-review when reviews are added.` and STOP (clean finish). (2) Remove Step 2a (Context Enrichment). (3) Remove Step 2b (Local Review Output Posting). (4) Update `--tool` flag valid values in Step 1.5 to `copilot` only — remove `codex` and `claude`; update validation error message to `Invalid tool: {name}. Valid values: copilot.`. (5) Update the `review_tools` config block shown in Phase 2 to show `copilot: true` only (remove `codex` and `claude` rows). (6) Update the skill description line to reflect Copilot-only automated review. (7) Remove `Bash(codex:*)` and `Bash(claude:*)` from `allowed-tools` frontmatter.

- [x] T005 [P] Update `src/kit/skills/fab-ff.md` — update Step 6 (Review) description to note that `_review.md` is the authoritative source of review behavior: add a parenthetical or note after "Dispatch `/fab-continue` as subagent — Review Behavior" clarifying "The subagent reads `_review.md` for review dispatch instructions — both inward and outward sub-agents are defined there."

- [x] T006 [P] Update `src/kit/skills/fab-fff.md` — same pointer update as T005: Step 6 (Review) dispatch description notes `_review.md` as authoritative source for inward + outward sub-agent dispatch.

## Phase 3: Integration & Config

- [x] T007 Update `fab/project/config.yaml` in this dev repo — remove `codex: true` and `claude: true` from the `review_tools` block, leaving only `copilot: true`. Result:
  ```yaml
  review_tools:
      copilot: true
  ```

- [x] T008 Create `src/kit/migrations/1.3.0-to-1.4.0.md` — migration file for removing `review_tools.codex` and `review_tools.claude` keys from user project configs. Structure: (1) Pre-check: confirm `fab/project/config.yaml` exists (skip if not); check if `review_tools` key exists (print skip message and stop if not). (2) Changes: `yq -i 'del(.review_tools.codex)' fab/project/config.yaml` then `yq -i 'del(.review_tools.claude)' fab/project/config.yaml`. (3) Verification: config does NOT contain `review_tools.codex`, does NOT contain `review_tools.claude`, DOES contain `review_tools.copilot` if the block is present. Migration is idempotent — `yq del` on absent keys is a no-op.

## Phase 4: Spec Files

<!-- These are [P] — all four spec files are independent of each other. Each task is conditional: verify the file exists before editing. -->

- [x] T009 [P] Update `docs/specs/skills/SPEC-fab-continue.md` — document `_review.md` delegation: (1) add `_review` row to the sub-agents table showing inward + outward sub-agents spawned in parallel; (2) update Review stage flow diagram/description to show dual parallel sub-agents dispatched via `_review.md`; (3) note that Review Behavior is now delegated to `_review.md` (single source of truth).

- [x] T010 [P] Update `docs/specs/skills/SPEC-git-pr-review.md` — document Copilot-only Phase 2: (1) update Configuration section to show `review_tools` with `copilot` key only (remove `codex` and `claude` rows); (2) update Phase 2 flow diagram to show Copilot-only path (remove Codex and Claude branches); (3) update Review Request Cascade table to show only Copilot; (4) remove Step 2a and Step 2b blocks; (5) update `--tool` valid values to `copilot` only.

- [x] T011 [P] Update `docs/specs/skills/SPEC-fab-ff.md` — note `_review.md` as authoritative review behavior source in Step 6 (Review) description and/or sub-agents table, consistent with the skill file update in T005.

- [x] T012 [P] Update `docs/specs/skills/SPEC-fab-fff.md` — same as T011: note `_review.md` as authoritative review behavior source in Step 6 (Review) description and/or sub-agents table, consistent with T006.

---

<!-- clarified: T003 says "add row" to the SRAD table but the review row already exists in fab-continue.md (line 75: `| review | [Review Behavior](#review-behavior) |`). The actual action is updating that existing row to reference `_review.md` rather than the inline anchor. Intent is unambiguous from spec context ("Stage dispatch table SHALL be updated to add a row: | review | Review Behavior (_review.md) |" means replace/update). Non-blocking. -->

## Execution Order

- T001 → T002 (read before writing the new file)
- T002 → T003, T004, T005, T006 (T002 creates `_review.md` which T003 references; T004–T006 are independent but should be written after T002 is settled)
- T003, T004, T005, T006 are [P] with respect to each other (different files)
- T007 and T008 are independent of T003–T006 and each other; may run alongside Phase 2
- T009, T010, T011, T012 are [P] with respect to each other; require T002–T006 done (to match the implementation they document)
