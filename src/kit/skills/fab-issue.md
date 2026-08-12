---
name: fab-issue
description: "Link the active change to Linear — find-or-create: search issues and projects for a semantic match against the intake, link an existing issue, create one in a matching project, or (with confirmation) create a user-assigned issue with no project. Gated on linear_workspace config and Linear MCP availability."
---

# /fab-issue [<change>] [<issue-id>]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Contents

- Purpose
- Arguments
- Pre-flight
- Gate Chain
- Behavior
- Created-Issue Content
- Autonomous Carve-Out
- Output
- Error Handling
- Key Properties

---

## Purpose

Link a change to Linear **after** intake, when a clean description and change type exist and matching quality is best. This is the push/search direction of Linear linking: search the user's Linear issues and projects for a semantic match against the change's intake, then link (find) or create-and-link (create). The pull direction — an explicit Linear ID handed to `/fab-new`/`/fab-draft` — is unchanged and owned by `_intake.md` Step 0.

Linking before ship matters: `/git-pr` renders the PR title as `{type}: {issues} {title}` from `fab status get-issues`, which drives Linear's PR-lifecycle auto-transitions. Linking after ship is still valid, but a late link does not update an already-created PR title.

---

## Arguments

Both optional, in any order — classified by value:

- **`<issue-id>`** — an argument matching `[A-Z]+-\d+` (e.g., `DEV-988`). Force-links that issue, skipping search entirely (Step 2).
- **`<change>`** — any other argument: target a specific change instead of the active one. Passed to preflight as `$1` (see `_preamble.md` §2; the override is transient).

---

## Pre-flight

Run `fab preflight [change]` per `_preamble.md` §2 (validates init, resolves the change, logs the command). Use its `name` field as `{name}` throughout. On preflight failure, STOP and surface stderr per the standard failure rule.

---

## Gate Chain

Evaluate these gates **in order** immediately after pre-flight. Each failed gate reports exactly one line and stops cleanly — never an error state, never a stage transition, never a search or creation:

1. **Idempotency guard** — run `fab status get-issues {name} --json`. If the array is non-empty: report `Already linked: {ids}` and STOP. This is the Constitution III re-run guard — a second run performs no Linear search and creates no second issue.
2. **Config gate** — `project.linear_workspace` unset or null in `fab/project/config.yaml`: report `Linear not configured (project.linear_workspace) — skipping` and STOP.
3. **MCP gate** — Linear MCP tools (`mcp__claude_ai_Linear__*`) unavailable in the session: report `Linear MCP unavailable — skipping` and STOP.

---

## Behavior

### Step 1: Load the Intake

Read `fab/changes/{name}/intake.md`. Extract:

- **Change human name** — the intake's first `# ` heading (strip any `Intake: ` prefix). This is the issue title source, NOT the folder slug.
- **Match query material** — the human name plus the `## Why` section and the `## What Changes` section (headings and summaries).
- **Why text** — the `## Why` section, condensed, for created-issue descriptions.

### Step 2: Explicit Issue-ID Path (argument provided)

Skips search entirely. Mirrors `_intake.md` Step 0's pull path for changes created before Linear linking was considered.

1. Fetch the issue via the Linear MCP (`mcp__claude_ai_Linear__get_issue`) to validate it exists.
2. **Fetch failure** (not found, API error): report `Linear issue {ID} not found — not linked` and STOP. Do NOT fall back to search; do NOT link.
3. On success: link via `fab status add-issue {name} {ID}` and report `Linked: {ID} — {title}`.

### Step 3: Search (no argument)

Using the match query material from Step 1, search Linear via the MCP: issue search scoped to the user's teams (`mcp__claude_ai_Linear__list_issues`), plus a project listing. Matching is agent semantic judgment over the results — compare intent and scope, not keywords. Only **non-completed, non-canceled** issues are link candidates.

### Step 4: Three-Branch Outcome

Take exactly one branch:

- **(a) Issue match** → link it: `fab status add-issue {name} {ID}`. Report `Linked: {ID} — {title}`.
- **(b) Project match, no issue match** → create an issue in that project via the MCP (content per § Created-Issue Content), then `fab status add-issue {name} {ID}`. Report `Created + linked: {ID} in project {project}`. No user confirmation — this branch proceeds unprompted.
- **(c) No match** → present the proposed issue (title / description / team) and **ask the user to confirm** before creating an issue with no project, assigned to the current user, then `add-issue`. In a promptless context this branch defers instead — see § Autonomous Carve-Out.

**Multiple plausible candidates** (several issues could match): present the candidates and let the user pick one or decline. Declining all candidates falls to branch (c). In a promptless context, multiple-candidate ambiguity resolves like branch (c) — defer per § Autonomous Carve-Out.

---

## Created-Issue Content

Issues created by branches (b) and (c) use:

| Field | Value |
|-------|-------|
| Title | The change's human-readable name (intake H1, not the folder slug) |
| Description | The intake's Why (condensed), ending with a `fab change: {folder-name}` reference line |
| Assignee | The current Linear user (MCP viewer) |
| Project | Set only in branch (b) — the matched project |
| Team | Branch (b): the matched project's team. Otherwise: the user's sole team; if the user has multiple teams, asked as part of branch (c)'s confirmation |

---

## Autonomous Carve-Out

When invoked from an orchestrator (e.g., `/fab-fff` Step 3.5) or any promptless context:

- Branch (c) — and the multiple-candidate ambiguity case — MUST NOT prompt. Report `No Linear match — issue creation deferred (run /fab-issue to create one)` and exit cleanly.
- Branches (a) and (b) proceed unprompted.

---

## Output

Report the outcome line from the branch taken (`Already linked: …`, the gate skip lines, `Linked: …`, `Created + linked: …`, or the deferral line), then end with the standard state-derived `Next:` line per `_preamble.md` § Next Steps Convention — the change's stage is unchanged by this skill, so the `Next:` state is whatever the change's progress map already says.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight fails | STOP, surface stderr (no change resolved, or project not initialized) |
| `issues` array non-empty | Gate 1 — report `Already linked: {ids}`, stop |
| `linear_workspace` unset/null | Gate 2 — report `Linear not configured (project.linear_workspace) — skipping`, stop |
| Linear MCP unavailable | Gate 3 — report `Linear MCP unavailable — skipping`, stop |
| Explicit-ID fetch fails | Report `Linear issue {ID} not found — not linked`, stop (no search fallback, no link) |
| No match, promptless context | Report `No Linear match — issue creation deferred (run /fab-issue to create one)`, exit cleanly |
| User declines creation (branch c) | Report no link made; no state touched |
| `fab status add-issue` fails | STOP, surface stderr per `_preamble.md` § failure rule |
| `intake.md` missing | STOP: `No intake.md found. Run /fab-continue intake to regenerate the intake first.` |

---

## Key Properties

| Property | Value |
|----------|-------|
| Requires active change? | Yes (or a `[change]` override) |
| Runs preflight? | Yes — `_preamble.md` §2 |
| Read-only? | No — `fab status add-issue` plus the Linear-side issue create are the only writes |
| Idempotent? | Yes — the Gate 1 `get-issues` guard makes re-runs report-and-stop |
| Advances stage? | No — runs no `fab status` transition command |
| Modifies `.fab-status.yaml`? | No |
| Run before ship? | Recommended — `/git-pr` renders linked IDs into the PR title; late links are valid but skip title-based automation |
| Prompts? | Interactive: branch (c) and multiple-candidate selection only. Promptless: never — defers instead |
