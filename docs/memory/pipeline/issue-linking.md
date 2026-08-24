---
type: memory
description: "`/fab-issue` — post-intake Linear find-or-create: the graceful gate chain (issues-array idempotency guard, linear_workspace config, Linear MCP), explicit-id force-link, three-branch find-or-create with created-issue content rules and the promptless deferral carve-out, fab-fff's optional inline Step 3.5 wiring, and the end-to-end Linear linking model (intake pull path, push path, /git-pr PR-title auto-linking)."
---
# Issue Linking

**Domain**: pipeline

## Overview

A change links to Linear issues through the `issues` array in its `.status.yaml` — the single linkage substrate, written by `fab status add-issue` and read by `fab status get-issues`. Three paths write or consume it:

- **Pull path (intake)** — owned by `_intake.md` Step 0: an explicit Linear ID (`[A-Z]+-\d+`) handed to `/fab-new` / `/fab-draft` is fetched via the Linear MCP (title, description, state, labels, branchName), recorded via `fab status add-issue` at change creation, and duplicate-guarded by a word-anchored `grep -lw` scan of existing `issues` arrays. See [planning-skills.md](/pipeline/planning-skills.md).
- **Push path (post-intake)** — `/fab-issue` (this file's subject): find-or-create from a natural-language intake, run after intake when a clean description and change type exist.
- **PR-title auto-linking (ship)** — `/git-pr` reads the array via `fab status get-issues` and renders the PR title as `{type}: {issues} {title}` (space-joined) when it is non-empty, driving Linear's PR-lifecycle issue-status transitions; the PR body's `## Meta` block renders the same IDs as an `**Issues**:` line (hyperlinked when `project.linear_workspace` is set). See [execution-skills.md](/pipeline/execution-skills.md). A link must land before ship to reach the title — a late link is valid but skips title-based automation.

Issue IDs never appear in folder slugs or branch names; they live only in the `issues` array — see [change-lifecycle.md](/pipeline/change-lifecycle.md) § Naming.

## The `/fab-issue` Skill

`src/kit/skills/fab-issue.md` — user-invocable, takes optional `[<change>] [<issue-id>]`, requires an active change (or the transient `[<change>]` override), advances no stage (runs no `fab status` transition command), and ends with the standard state-derived `Next:` line. It declares no `helpers:` (loads only `_preamble`) and maps to the `Completion` help group in `fab_help.go`. Its only writes are `fab status add-issue` and the Linear-side issue create.

### Gate Chain

After preflight, three gates evaluate in order; each failed gate reports exactly one line and stops cleanly — never an error state, never a transition, never a search or creation:

1. **Idempotency guard** — `fab status get-issues {name} --json` non-empty ⇒ report `Already linked: {ids}` and stop. This is the Constitution III re-run guard: a second run performs no search and creates no second issue.
2. **Config gate** — `project.linear_workspace` unset/null in `fab/project/config.yaml` ⇒ report `Linear not configured (project.linear_workspace) — skipping` and stop.
3. **MCP gate** — Linear MCP tools (`mcp__claude_ai_Linear__*`) unavailable in the session ⇒ report `Linear MCP unavailable — skipping` and stop.

### Explicit-ID Force-Link

`/fab-issue <issue-id>` skips search entirely: fetch the issue via the Linear MCP to validate it exists — a fetch failure reports `Linear issue {ID} not found — not linked` and stops (no search fallback, no link) — then link via `fab status add-issue {name} {ID}`. This mirrors the `_intake` pull path for changes created before linking was considered.

### Three-Branch Find-or-Create

With no argument, the skill builds the match query from the intake (H1 change name + `## Why` + `## What Changes`), searches the user's Linear issues (scoped to their teams) and projects via MCP — matching is agent semantic judgment over intent and scope, and only non-completed, non-canceled issues qualify — then takes exactly one branch:

- **(a) Issue match** → link it (`add-issue`); report `Linked: {ID} — {title}`.
- **(b) Project match, no issue match** → create an issue in that project, then link; report `Created + linked: {ID} in project {project}`. This branch proceeds without user confirmation.
- **(c) No match** → present the proposed issue (title / description / team) and ask the user to confirm before creating an issue with no project, assigned to the current user. Multiple plausible candidates are presented for the user to pick or decline; declining all falls to this branch.

### Created-Issue Content

| Field | Value |
|-------|-------|
| Title | The change's human-readable name (intake H1, not the folder slug) |
| Description | The intake's Why (condensed), ending with a `fab change: {folder-name}` reference line |
| Assignee | The current Linear user (MCP viewer) |
| Project | Set only in branch (b) — the matched project |
| Team | Branch (b): the matched project's team. Otherwise: the user's sole team; if the user has multiple teams, asked as part of branch (c)'s confirmation |

### Promptless Carve-Out

When invoked from an orchestrator or any promptless context, branch (c) — and the multiple-candidate ambiguity case — MUST NOT prompt: report `No Linear match — issue creation deferred (run /fab-issue to create one)` and exit cleanly. Branches (a) and (b) proceed unprompted.

## Orchestrator Wiring

`/fab-fff` runs the `/fab-issue` behavior as an optional **Step 3.5: Link Linear Issue** between the bracket handoff and Step 4 (Ship), **inline in the orchestrator's context in both lanes** — no dispatch, no `fab resolve-agent`, no `.status.yaml` progress entry — and skips it when `progress.ship` is `done` (the PR title has already shipped). All gates and the promptless carve-out apply, so an unconfigured project sees zero behavior change and a skip or deferral never blocks ship. `/fab-proceed` inherits the step via its `/fab-fff` delegation; `/fab-ff` is deliberately not wired (it ends at hydrate — ship happens later via `/git-pr`), so its users run `/fab-issue` manually.

## Design Decisions

### Standalone Post-Intake Skill
**Decision**: Linear find-or-create is a separate user-invocable skill (`/fab-issue`), not embedded in `/fab-new` / `/fab-draft` / `_intake`.
**Why**: Matching quality is best post-intake (a clean description and change type exist); external side effects do not belong in intake (Constitution III); `/fab-draft` can be invoked in bulk to queue intakes, so auto-creating Linear issues per draft would spam the workspace.
**Rejected**: Embedding in intake (side effects + bulk-draft spam); linking at ship time inside `/git-pr` (too late for the user to steer, and `/git-pr` is deliberately prompt-free/autonomous).
*Introduced by*: 260812-z5qt-linear-issue-find-or-create

### Inline Execution of the fab-fff Link Step
**Decision**: Step 3.5 runs inline in the orchestrator's context in BOTH lanes, with no `fab resolve-agent` call.
**Why**: MCP tool availability is session-bound and the step is a handful of MCP calls plus one `fab status add-issue`; it is an optional linking action, not a pipeline stage with a role profile, and a dispatch would cost a cold start to save nothing.
**Rejected**: Dispatching it as a stage worker — it has no stage, no result-file contract, and no `.status.yaml` progress entry to transition.
*Introduced by*: 260812-z5qt-linear-issue-find-or-create

### Help Group: Completion
**Decision**: `/fab-issue` maps to the `Completion` help group.
**Why**: It sits with the ship-adjacent lifecycle commands (`git-pr`, `git-pr-review`) whose PR-title mechanism it feeds; it is not a change-creation or planning command.
**Rejected**: `Start & Navigate` (it does not create or switch changes) and `Planning` (it generates no artifacts).
*Introduced by*: 260812-z5qt-linear-issue-find-or-create
