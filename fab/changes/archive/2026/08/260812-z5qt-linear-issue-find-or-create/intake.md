# Intake: Linear Issue Find-or-Create Skill

**Change**: 260812-z5qt-linear-issue-find-or-create
**Created**: 2026-08-12

## Origin

Conversational — designed in a `/fab-discuss` session, then created via `/fab-new`.

> New /fab-issue skill — Linear find-or-create linking step between intake and pipeline: search the user's Linear projects and issues for a match against the active change's intake; if an issue matches, link it via fab status add-issue; if only a project matches, create an issue in that project and link it; if nothing matches, confirm with the user then create an unassigned issue (linked to the user, no project) and link it. Gated on linear_workspace config + Linear MCP availability; idempotent via the existing issues array guard; must run before ship so /git-pr picks up the ID in the PR title. Orchestrators (fab-proceed, optionally fab-fff) may call it as an optional step.

Key decisions from the discussion (user confirmed all):

- A **separate skill**, NOT embedded in `/fab-new`/`/fab-draft`/`_intake` — matching quality is better post-intake (clean description + change type exist), external side effects don't belong in intake (Constitution III), and `/fab-draft` is a bulk consumer via `/fab-dedupe` (auto-creating Linear issues per draft would spam the workspace).
- The **pull direction already exists** and is untouched: `_intake.md` Step 0 parses an explicit Linear ID, fetches via MCP, records via `fab status add-issue`, and has a `grep -lw` collision scan. This change adds only the **push/search direction** (natural-language intake → find-or-create).
- **Branch names stay ID-free** (deliberate existing design — `docs/memory/pipeline/change-lifecycle.md`). Linear auto-linking rides the PR title: `/git-pr` renders `{type}: {issues} {title}` from `fab status get-issues` (`docs/specs/naming.md`), so PR-lifecycle → issue-status transitions work once the ID is in the `issues` array before ship. Nothing in the PR path changes.
- The no-match branch **confirms with the user before creating** (external state from a fuzzy non-match); the project-match branch creates without confirmation (user-specified policy).

## Why

1. **Pain point**: fab changes link to Linear only when the user hands an explicit issue ID to `/fab-new`. Natural-language changes — the common case — produce no Linear linkage at all, so work happens invisibly to Linear projects, and existing matching issues stay un-linked and un-progressed.
2. **Consequence if unfixed**: Linear boards drift from reality; duplicate issues get filed manually after the fact; PR-merge auto-transitions (already wired via the PR title) never fire because the `issues` array stays empty.
3. **Why this approach**: a standalone post-intake skill reuses the entire existing linkage substrate (`issues` array, `fab status add-issue`/`get-issues`, `/git-pr` title rendering, `linear_workspace` config) — zero schema or Go CLI surface changes beyond the help catalog. Alternatives rejected: embedding in intake (side effects + bulk-draft spam, above); doing it at ship time inside `/git-pr` (too late for the user to steer, and `/git-pr` is deliberately prompt-free/autonomous).

## What Changes

### 1. New skill: `src/kit/skills/fab-issue.md` (`/fab-issue`)

A user-invocable skill that links the **active change** to Linear. Flow:

1. **Preflight + gate** (in order, each exits gracefully with a one-line report — never an error):
   - `fab preflight` resolves the active change (standard Change Context loading; accepts the usual optional `[change]` override argument).
   - **Idempotency guard**: `fab status get-issues {name} --json` — if non-empty, report `Already linked: {ids}` and STOP. This is the Constitution III re-run guard; no re-search, no second issue.
   - **Config gate**: `project.linear_workspace` unset/null in `fab/project/config.yaml` → report `Linear not configured (project.linear_workspace) — skipping` and STOP.
   - **MCP gate**: Linear MCP tools unavailable in the session → report `Linear MCP unavailable — skipping` and STOP.
2. **Optional explicit argument**: `/fab-issue DEV-988` skips search entirely — fetch the issue via MCP to validate it exists, then link (`fab status add-issue`) and report. Mirrors `_intake.md`'s pull path for changes created before Linear linking was considered.
3. **Search** (no argument): build the match query from the intake — change human name + `## Why` + `## What Changes` headings/summaries. Search Linear via the MCP (issue search scoped to the user's teams, plus project listing). Matching is agent semantic judgment over the results; only **non-completed, non-canceled** issues are link candidates.
4. **Three-branch outcome**:
   - **Issue match** → `fab status add-issue {name} {ID}`; report `Linked: {ID} — {title}`.
   - **Project match, no issue match** → create an issue in that project via MCP (content per below), then `add-issue`; report `Created + linked: {ID} in project {project}`.
   - **No match** → present the proposed issue (title/description/team) and **ask the user to confirm** before creating an issue with no project, assigned to the user; then `add-issue`. Multiple plausible matches at step 3 are resolved the same way — present candidates, let the user pick or decline.
5. **Created-issue content**: title = change human name (intake H1, not the folder slug); description = the intake's Why (condensed) + a `fab change: {folder-name}` reference line; assignee = the current Linear user (MCP viewer); project only in the project-match branch; team = the matched project's team, else the user's sole team, else asked as part of the no-match confirmation.
6. **Output** ends with the standard state-derived `Next:` line (the change's current stage is unchanged — this skill advances nothing).

**Autonomous carve-out**: when invoked from an orchestrator (or any promptless context), the no-match branch does NOT prompt — it reports `No Linear match — issue creation deferred (run /fab-issue to create one)` and exits cleanly. Issue-match and project-match branches proceed unprompted (user-specified policy).

**Key properties**: read-only except `add-issue` + the Linear-side create; idempotent (guard above); advances no stage; requires an active change (or `[change]` override); must run **before ship** to affect the PR title — but linking after ship is still valid (Linear picks up edited PR titles is NOT assumed; late links simply skip title-based automation).

### 2. Orchestrator wiring: one optional step in `/fab-fff` before ship

`src/kit/skills/fab-fff.md` gains a single optional pre-ship step: invoke the `/fab-issue` behavior (gated exactly as above — all three gates skip silently, and the autonomous carve-out applies, so an unconfigured project sees zero behavior change). `/fab-proceed` inherits this for free by delegating to `/fab-fff` — no edit to `fab-proceed.md`'s flow beyond a pointer if its prose enumerates fab-fff's steps. `/fab-ff` stops at hydrate (ship happens later via `/git-pr`), so it is deliberately NOT wired — users on that flow run `/fab-issue` manually. Twin-sweep obligation: `fab-ff` ↔ `fab-fff` contrastive prose must be swept for the new asymmetry.

### 3. Catalog + docs updates

- `src/go/fab/cmd/fab/fabhelp.go` — add `"fab-issue"` to the category map (likely "Start & Navigate" or the category holding `/git-pr`) + `fabhelp_test.go` update (Go change ⇒ tests, per constitution).
- `docs/specs/skills.md` — per-skill section for `/fab-issue` + helper-matrix row.
- `docs/specs/user-flow.md` — command-map entry.
- `docs/specs/glossary.md` — `/fab-issue` entry; sweep aggregate specs per the sibling-sweep class.
- No `_cli-fab.md` change (no fab CLI signature changes — the skill reuses `fab status add-issue`/`get-issues` verbatim).
- No migration (no user-data restructuring; `linear_workspace` config field already exists — added in 0.37.0).

### Non-goals

- No Linear status pushes at link time (PR auto-linking owns state transitions).
- No issue IDs in branch names or change slugs (existing design retained).
- No backfill sweep over existing/archived changes.
- No changes to `_intake.md`'s pull path or `/git-pr`.

## Affected Memory

- `pipeline/issue-linking`: (new) — the `/fab-issue` skill: gates, three-branch find-or-create, idempotency guard, autonomous carve-out, and the end-to-end Linear linking model (pull path + push path + PR-title auto-linking)
- `pipeline/planning-skills`: (modify) — pointer from the `_intake` Linear pull path to the new post-intake push path
- `pipeline/execution-skills`: (modify) — fab-fff's optional pre-ship step; cross-ref from `/git-pr`'s `get-issues` consumption

## Impact

- **New**: `src/kit/skills/fab-issue.md`
- **Modified**: `src/kit/skills/fab-fff.md`, `src/go/fab/cmd/fab/fabhelp.go`, `src/go/fab/cmd/fab/fabhelp_test.go`, `docs/specs/skills.md`, `docs/specs/user-flow.md`, `docs/specs/glossary.md` (+ any aggregate-spec sweep hits)
- **Runtime dependencies**: Linear MCP (session-level, optional — all behavior gates on availability); `project.linear_workspace` config field (existing)
- **Test surface**: `fabhelp_test.go` only — the skill itself is markdown (Pure Prompt Play, Constitution I)

## Open Questions

- (none — the design discussion resolved all decision points; see Assumptions)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Standalone skill named `/fab-issue`, not embedded in intake skills | Discussed — user agreed to the separate-skill recommendation and its reasons (post-intake matching, no intake side effects, draft bulk safety) | S:90 R:70 A:90 D:85 |
| 2 | Certain | Three-branch policy: issue match → link; project match → create in project; no match → confirm then create user-assigned/no-project | User's own specification, verbatim | S:95 R:75 A:85 D:90 |
| 3 | Certain | Gate on `linear_workspace` + Linear MCP availability; all gates skip gracefully, never error | User's specification; config field already exists for exactly this opt-in | S:85 R:80 A:85 D:85 |
| 4 | Certain | Idempotency guard: non-empty `issues` array → report and stop | User's specification + Constitution III; `get-issues --json` is the existing probe | S:80 R:85 A:90 D:85 |
| 5 | Confident | Match input = intake H1 + Why + What Changes; semantic match via agent judgment over MCP search; only non-completed/non-canceled issues qualify | Discussed — post-intake matching was the stated reason for the skill's position; state filter is the one obvious default | S:60 R:85 A:75 D:70 |
| 6 | Confident | Wire only `/fab-fff` (pre-ship, optional); `/fab-proceed` inherits via delegation; `/fab-ff` deliberately unwired | User said "fab-proceed, optionally fab-fff"; fab-proceed delegates to fab-fff so one wiring point covers both; fab-ff has no ship stage | S:65 R:75 A:70 D:65 |
| 7 | Confident | Autonomous carve-out: promptless contexts defer no-match creation instead of prompting; match branches proceed | fab-fff runs unattended post-gate — a blocking prompt would violate its 0-interruption budget; deferring external-state creation is the conservative default | S:55 R:70 A:60 D:60 |
| 8 | Confident | Created-issue content: title = change name, description = condensed Why + change reference, assignee = viewer, team = project's team / sole team / asked at confirmation | Standard find-or-create shape; team is required by Linear and the no-match branch already has a confirmation prompt to ride | S:50 R:88 A:70 D:70 |
| 9 | Confident | Optional explicit `<issue-id>` argument force-links, skipping search | Not user-requested but cheap, reversible, and mirrors the existing `_intake` pull path for late linking | S:35 R:80 A:55 D:50 |
| 10 | Confident | Go surface limited to the `fabhelp.go` catalog entry (+ test); no `_cli-fab.md` change, no migration | Skill reuses existing `fab status` verbs verbatim; no CLI signatures change; no user data restructured | S:70 R:80 A:85 D:80 |

10 assumptions (4 certain, 6 confident, 0 tentative, 0 unresolved).
