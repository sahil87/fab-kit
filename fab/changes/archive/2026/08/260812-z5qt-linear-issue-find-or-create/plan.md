# Plan: Linear Issue Find-or-Create Skill

**Change**: 260812-z5qt-linear-issue-find-or-create
**Intake**: `intake.md`

## Requirements

### Skill: `/fab-issue` gate chain

#### R1: Skill file with graceful gate chain
A new user-invocable skill `src/kit/skills/fab-issue.md` SHALL exist. On invocation it MUST evaluate, in order: (1) `fab preflight [change]` resolves a change (standard optional `[change]` override); (2) the idempotency guard (R2); (3) `project.linear_workspace` is set and non-null in `fab/project/config.yaml`; (4) Linear MCP tools are available in the session. Each failed gate MUST produce a one-line report and stop cleanly — never an error state, never a stage transition.

- **GIVEN** a project with `linear_workspace` unset
- **WHEN** `/fab-issue` runs
- **THEN** it reports `Linear not configured (project.linear_workspace) — skipping` and stops without touching any state

#### R2: Idempotency guard on the `issues` array
The skill MUST run `fab status get-issues {name} --json` before any search or creation. If the array is non-empty, it SHALL report `Already linked: {ids}` and stop.

- **GIVEN** a change whose `.status.yaml` `issues` array contains `DEV-988`
- **WHEN** `/fab-issue` runs a second time
- **THEN** no Linear search or creation occurs and the output reports the existing link (Constitution III)

#### R3: Explicit issue-id argument force-links
`/fab-issue <issue-id>` (matching `[A-Z]+-\d+`) SHALL skip search: fetch the issue via the Linear MCP to validate it exists, then link via `fab status add-issue {name} {ID}`. A fetch failure reports and stops without linking.

- **GIVEN** an active change with no linked issues
- **WHEN** the user runs `/fab-issue DEV-988` and the issue exists
- **THEN** `DEV-988` is recorded in the `issues` array and reported as linked

### Skill: find-or-create matching

#### R4: Three-branch find-or-create
With no argument, the skill SHALL build a match query from the intake (H1 change name + `## Why` + `## What Changes`), search the user's Linear issues and projects via MCP, and take exactly one branch: (a) **issue match** (semantic match, issue not completed/canceled) → link it; (b) **project match, no issue match** → create an issue in that project, then link; (c) **no match** → present the proposed issue and ask the user to confirm before creating an issue with no project, assigned to the current user, then link. Multiple plausible issue candidates are presented for the user to pick or decline.

- **GIVEN** a Linear workspace containing an open issue that describes the same work as the intake
- **WHEN** `/fab-issue` runs
- **THEN** that issue's ID lands in the `issues` array and no new issue is created

- **GIVEN** a workspace with a topically matching project but no matching issue
- **WHEN** `/fab-issue` runs
- **THEN** a new issue is created inside that project and linked

#### R5: Created-issue content
Created issues SHALL use: title = the change's human-readable name (intake H1); description = the intake's Why (condensed) plus a `fab change: {folder-name}` reference line; assignee = the current Linear user (MCP viewer); project set only in branch (b); team = the matched project's team in branch (b), else the user's sole team, else asked as part of branch (c)'s confirmation.

- **GIVEN** branch (b) fires for project P owned by team T
- **WHEN** the issue is created
- **THEN** it belongs to project P and team T, is assigned to the current user, and its description ends with the `fab change:` reference line

#### R6: Autonomous carve-out
When invoked from an orchestrator or any promptless context, branch (c) MUST NOT prompt — it reports `No Linear match — issue creation deferred (run /fab-issue to create one)` and exits cleanly. Branches (a) and (b) proceed unprompted.

- **GIVEN** `/fab-fff` runs the optional link step with no matching issue or project
- **WHEN** the no-match branch is reached
- **THEN** nothing is created, the deferral line is reported, and the pipeline continues

### Orchestrator: `/fab-fff` wiring

#### R7: Optional pre-ship link step in fab-fff only
`src/kit/skills/fab-fff.md` SHALL gain a **Step 3.5: Link Linear Issue (optional)** between the bracket handoff and Step 4 (Ship): run the `/fab-issue` behavior **inline in both lanes** (no dispatch, no `fab resolve-agent` — it is an optional linking action, not a pipeline stage), with all R1 gates and the R6 carve-out applying, so an unconfigured project sees zero behavior change and a gate skip never blocks ship. `/fab-ff` is deliberately NOT wired (it ends at hydrate). `/fab-proceed` inherits via its `/fab-fff` delegation.

- **GIVEN** a project with no `linear_workspace`
- **WHEN** `/fab-fff` reaches Step 3.5
- **THEN** the step reports the skip in one line and Step 4 (Ship) proceeds unchanged

### Catalog & docs

#### R8: Help catalog + tests
`src/go/fab/cmd/fab/fabhelp.go` `skillToGroupMap` SHALL gain `"fab-issue": "Completion"`, and `fabhelp_test.go`'s `expectedMapped` list SHALL include it. The affected Go test package MUST pass.

- **GIVEN** the updated binary and a synced kit
- **WHEN** `fab fab-help` renders
- **THEN** `/fab-issue` appears under the Completion group with its frontmatter description

#### R9: Spec docs updated
`docs/specs/skills.md` SHALL gain a `/fab-issue` per-skill section; `docs/specs/user-flow.md`'s command map and `docs/specs/glossary.md` SHALL gain matching entries. The sibling-sweep obligation applies: grep aggregate specs and the `fab-ff`↔`fab-fff` twin prose for claims the new step makes stale (e.g., enumerations of fab-fff's steps).

- **GIVEN** the shipped change
- **WHEN** a reader greps `fab-issue` across `docs/specs/`
- **THEN** skills.md, user-flow.md, and glossary.md each describe it consistently with the skill file

### Non-Goals

- No Linear status pushes at link time — PR-title auto-linking owns state transitions
- No issue IDs in branch names or change slugs (existing design retained)
- No backfill over existing/archived changes; no `_intake.md` pull-path or `/git-pr` changes
- No new config fields, no `_cli-fab.md` change (no fab CLI signatures change), no migration

### Design Decisions

#### Inline execution of the fab-fff link step
**Decision**: Step 3.5 runs inline in the orchestrator's context in BOTH lanes, with no `fab resolve-agent` call.
**Why**: MCP tool availability is session-bound and the step is a handful of MCP calls plus one `fab status add-issue`; it is an optional linking action, not a pipeline stage with a role profile, and a dispatch would cost a cold start to save nothing.
**Rejected**: Dispatching it as a stage worker — it has no stage, no result-file contract, and no `.status.yaml` progress entry to transition.
*Introduced by*: 260812-z5qt-linear-issue-find-or-create

#### Help group: Completion
**Decision**: `/fab-issue` maps to the `Completion` help group.
**Why**: It sits with the ship-adjacent lifecycle commands (`git-pr`, `git-pr-review`) whose PR-title mechanism it feeds; it is not a change-creation or planning command.
**Rejected**: `Start & Navigate` (it does not create or switch changes) and `Planning` (it generates no artifacts).
*Introduced by*: 260812-z5qt-linear-issue-find-or-create

## Tasks

### Phase 1: Setup

- [x] T001 Add `"fab-issue": "Completion"` to `skillToGroupMap` in `src/go/fab/cmd/fab/fabhelp.go` <!-- R8 -->
- [x] T002 Add `"fab-issue"` to `expectedMapped` in `src/go/fab/cmd/fab/fabhelp_test.go` and run `go test ./src/go/fab/cmd/fab/ -run TestFabHelp` <!-- R8 -->

### Phase 2: Core Implementation

- [x] T003 Write `src/kit/skills/fab-issue.md` — frontmatter (name/description, no `helpers:`), preamble pointer, Arguments (`[<change>] [<issue-id>]`), gate chain (R1/R2), explicit-id path (R3), search + three branches (R4), created-issue content rules (R5), autonomous carve-out (R6), Output with state-derived `Next:` line, Error Handling table, Key Properties table <!-- R1 -->

### Phase 3: Integration & Edge Cases

- [x] T004 Add **Step 3.5: Link Linear Issue (optional)** to `src/kit/skills/fab-fff.md` between the bracket handoff prose and Step 4, running the `/fab-issue` behavior inline in both lanes with a skip-never-blocks-ship note; update the file's Contents/step framing lines that enumerate "Steps 4–5" where the new step changes them <!-- R7 -->
- [x] T005 Sweep: grep `Steps 4–5\|fab-ff\b` twin/aggregate prose (`src/kit/skills/fab-ff.md`, `_pipeline.md`, `docs/specs/skills.md`, `docs/specs/overview.md`, `docs/specs/architecture.md`, `docs/specs/glossary.md`) for enumerations of fab-fff's post-hydrate steps or "issues array is written only at intake" claims; update every stale occurrence <!-- R9 -->

### Phase 4: Polish

- [x] T006 [P] Add `/fab-issue` per-skill section to `docs/specs/skills.md` (+ helper-matrix row if the file carries one) <!-- R9 -->
- [x] T007 [P] Add `/fab-issue` to `docs/specs/user-flow.md` command map <!-- R9 -->
- [x] T008 [P] Add `/fab-issue` glossary entry to `docs/specs/glossary.md` <!-- R9 -->

## Execution Order

- T003 blocks T004 (the fab-fff step references the finished skill's gate list)
- T005 runs after T004 (sweep sees the final wording); T006–T008 are independent

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/fab-issue.md` exists with the four-gate chain in the specified order, each gate reporting and stopping cleanly
- [x] A-002 R2: The `get-issues --json` guard precedes all search/create logic and stops on non-empty
- [x] A-003 R3: The explicit-id path validates via MCP fetch before `add-issue` and never searches
- [x] A-004 R4: All three branches are specified with their exact link/create/confirm actions
- [x] A-005 R5: Created-issue content rules (title/description/assignee/project/team) are stated verbatim in the skill
- [x] A-006 R6: The promptless carve-out defers branch (c) with the exact deferral line and lets (a)/(b) proceed
- [x] A-007 R7: fab-fff.md carries Step 3.5 inline-both-lanes with gates, carve-out, and skip-never-blocks-ship
- [x] A-008 R8: fabhelp.go maps fab-issue to Completion; fabhelp_test.go covers it; the test package passes

### Behavioral Correctness

- [x] A-009 R7: Step 3.5 failure/skip paths cannot prevent Step 4 (Ship) from running
- [x] A-010 R1: No `fab status` transition command appears anywhere in the new skill (it advances no stage)

### Scenario Coverage

- [x] A-011 R4: The skill documents the multiple-candidate case (present, user picks or declines)
- [x] A-012 R3: The fetch-failure case reports and stops without linking

### Edge Cases & Error Handling

- [x] A-013 R1: MCP-unavailable and workspace-unset gates each have a distinct one-line report
- [x] A-014 R5: The no-project team fallback chain (project's team → sole team → ask) is fully specified

### Code Quality

- [x] A-015 Pattern consistency: fab-issue.md follows the established skill-file shape (Contents, Arguments, Behavior, Output, Error Handling, Key Properties) and states owned rules OR points at owners, never both
- [x] A-016 No unnecessary duplication: gate/branch rules live once in fab-issue.md; fab-fff.md Step 3.5 points at them rather than restating
- [x] A-017 Canonical source only: all skill edits land in `src/kit/skills/`, never `.claude/skills/`
- [x] A-018 CLI ⇒ docs + tests: the Go change ships its test update; no `_cli-fab.md` change is needed because no CLI signature changes

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Help group = Completion | Ship-adjacent lifecycle command, alongside git-pr/git-pr-review | S:60 R:90 A:70 D:65 |
| 2 | Confident | Step 3.5 runs inline in both lanes, no resolve-agent | Session-bound MCP + optional action, not a stage; see Design Decisions | S:55 R:80 A:75 D:65 |
| 3 | Confident | New skill declares no `helpers:` (loads only `_preamble`) | It generates no artifacts and runs no SRAD/review procedures | S:65 R:90 A:85 D:80 |
| 4 | Certain | Skill advances no stage and ends with the state-derived `Next:` line | `_preamble.md` § Next Steps Convention; intake R-list specifies non-advancing | S:85 R:85 A:90 D:85 |

4 assumptions (1 certain, 3 confident, 0 tentative).
