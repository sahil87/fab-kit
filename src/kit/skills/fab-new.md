---
name: fab-new
description: "Start a new change — creates the intake, activates it, and creates the git branch."
helpers: [_generation, _srad, _intake]
---

# /fab-new <description>

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

## Contents

- Pre-flight
- Arguments
- Behavior
- Output
- Error Handling
- Key Properties

---

## Pre-flight

1. Verify `fab/project/config.yaml` and `fab/project/constitution.md` exist
2. **If either missing, STOP**: `fab/ is not initialized. Run /fab-setup first to bootstrap the project.`

---

## Arguments

- **`<description>`** *(required)* — natural language, Linear ticket ID (`DEV-988`), or backlog ID (`90g5`)

If no description: ask *"What change do you want to make?"*

---

## Behavior

### Steps 0–9: Create the Intake

Read `.claude/skills/_intake/SKILL.md` and execute the **Create-Intake Procedure** (Steps 0–9 — parse input, slug, gap analysis, create change, conversation context mining, generate `intake.md`, verify change type, confidence, SRAD question selection, advance to ready) with:

- **`{questioning-mode} = interactive`** — Step 8 asks the user via SRAD (no fixed cap; conversational mode when 5+ Unresolved). This is `/fab-new`'s existing intake-creation behavior.

The procedure advances intake to `ready` at its Step 9 and stops. `/fab-new`'s own tail (Steps 10–11 below) then activates the change and creates the git branch — these are `/fab-new`-specific and stay here; they are NOT part of the shared procedure.

### Step 10: Activate Change

After advancing intake to ready, activate the newly created change:

```bash
fab change switch "{name}"
```

Display the switch confirmation from stdout. This makes the change immediately available for `/fab-continue` without requiring a separate `/fab-switch` step.

> **Note**: For create-without-activate behavior (e.g., queuing a change for later), use `/fab-draft` instead.

### Step 11: Create Git Branch

After activating the change, create or check out the matching git branch inline.

First, verify the working directory is inside a git repository:

```bash
git rev-parse --is-inside-work-tree >/dev/null 2>&1
```

If this fails: warn `Not in a git repository — skipping branch creation` and continue. The change remains activated. This step is **non-fatal**.

If inside a git repo, read the context the conditions below use:

```bash
git branch --show-current                                                                  # current branch name
git status --porcelain | grep -v "fab/changes/{name}/" | wc -l                             # {dirty_count} — for the non-blocking note below
git rev-parse --verify "{name}" >/dev/null 2>&1                                            # does the target branch exist locally?
git rev-parse --verify "origin/{name}" >/dev/null 2>&1                                     # does it exist on the remote?
upstream=$(git config "branch.$(git branch --show-current).remote" 2>/dev/null || true)    # empty = local-only branch
fab resolve --folder "$(git branch --show-current)" --or-none                              # which change (if any) does the current branch belong to? "(none)" = none
```

> **fab-new-specific `{dirty_count}` derivation**: the porcelain count excludes `fab/changes/{name}/` — this change's own just-created artifacts (`intake.md`, `.status.yaml`, `.history.jsonl`) always exist uncommitted by Step 11, so counting them would fire the dirty-tree note on every run. Only *pre-existing* uncommitted work should trigger the note.

<!-- Keep this table in sync with git-branch.md Step 4 — same cases, same commands, same report strings (incl. the rename guard, the remote-only --track case, and the dirty-tree note). Two deliberate divergences: fab-new derives {dirty_count} excluding fab/changes/{name}/ (see the derivation note above) while git-branch counts the full porcelain output; and fab-new's rename-guard probe is the token-branching `fab resolve --folder … --or-none` (`(none)` vs a folder name — 260720-dow0) while git-branch keeps the strict exit-code form `fab change resolve … 2>/dev/null` (not migrated — its bare no-argument resolution is a hard stop by design). -->

**Evaluate in order, first match wins:**

| # | Condition | Command | Report |
|---|-----------|---------|--------|
| 1 | Current branch equals `{name}` | *(none)* | `Branch: {name} (already active)` |
| 2 | Target branch `{name}` exists locally (`git rev-parse --verify "{name}"` succeeds) | `git checkout "{name}"` | `Branch: {name} (checked out)` |
| 3 | Target exists only on the remote (`origin/{name}` verify succeeds) — do NOT recreate a divergent local | `git checkout --track "origin/{name}"` | `Branch: {name} (checked out, tracking origin/{name})` |
| 4 | On `main` or `master` | `git checkout -b "{name}"` | `Branch: {name} (created)` |
| 5 | Local-only branch (`upstream` empty) AND the **rename guard** passes: the current branch belongs to no change (the probe prints `(none)` — e.g., a disposable `wt create` name) or resolves to this SAME change (e.g., a worktree placeholder named with the change's own ID) | `git branch -m "{name}"` | `Branch: {name} (renamed from {old_branch})` |
| 6 | Local-only branch belonging to a different change (the probe prints another change's folder — e.g., after `/fab-switch`; do NOT rename it away, caveat: the new branch inherits the old change's HEAD) OR pushed branch (`upstream` non-empty) | `git checkout -b "{name}"` | `Branch: {name} (created, leaving {old_branch} intact)` |

> **Dirty-tree note** (non-blocking — never prompt, never stash): when `{dirty_count}` > 0 AND the matched row runs `git checkout -b` or `git branch -m`, the uncommitted work rides onto the new branch. Append to the report line: ` — note: {dirty_count} uncommitted change(s) carried over from {old_branch}`.

If any git operation fails (e.g., uncommitted conflicts blocking checkout):
- Report the git error message
- The change remains activated
- Append: `Run /git-branch to create the branch manually`

---

## Output

```
{if Linear: "Fetching Linear issue DEV-988...\n"}
{if backlog: "Reading fab/backlog.md for [90g5]...\nFound: DEV-988 ...\n"}
Created fab/changes/{name}/

## Intake: {Change Name}

{intake content}

Intake complete.

Confidence: {score} / 5.0 ({N} decisions, cover: {cover})

Activated: {name}
Branch: {name} (created|created, leaving {old_branch} intact|checked out|checked out, tracking origin/{name}|renamed from {old_branch}|already active)[ — note: {dirty_count} uncommitted change(s) carried over from {old_branch}]

{if assumptions: "## Assumptions\n\n| # | Grade | Decision | Rationale | Scores |\n..."}

Next: {per state table — intake state (no activation preamble)}
```

> The Assumptions summary is the **final content block immediately before `Next:`** (per `_srad.md` § Assumptions Summary Block); omit it from the output only when 0 assumptions were made (the artifact's `## Assumptions` section is always present regardless).

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Config/constitution missing | Abort: "Run /fab-setup first." |
| No description | Ask for one |
| Intake template missing | Abort: "Kit may be corrupted." |
| `fab change new` collision (`Change ID already in use` — backlog IDs only; Linear collisions are caught by Step 3's issues-array scan) | Route to resume: point to `/fab-switch {existing-name}` then `/fab-continue` — do not retry creation (Step 3's collision check normally catches this first) |
| `fab change new` failure (other) | Surface stderr output to user and stop |
| `fab change switch` failure (Step 10) | Surface stderr output to user; intake is already at `ready` — user can manually run `/fab-switch {name}` to activate |
| Not in a git repo (Step 11) | Warn and skip branch creation — change is still activated |
| `git checkout` / `git branch` failure (Step 11) | Report the git error; change remains activated — user can run `/git-branch` manually |
| Linear ticket not found / API error | Warn, treat as natural language |
| Backlog ID not found | Abort with guidance |
| `fab/backlog.md` missing | Abort: "Use natural language or Linear ID instead." |

---

## Key Properties

| Property | Value |
|----------|-------|
| Idempotent? | Partially — re-running with the same backlog/Linear ID routes to resume (`/fab-switch {name}` + `/fab-continue`) instead of creating a duplicate; a natural-language re-run intentionally creates a new change each run |
| Advances stage? | Yes — intake to `ready` |
| Modifies `.fab-status.yaml`? | Yes — activates the new change (Step 10) |
| Modifies git state? | Yes — creates/checks out the change branch (Step 11, non-fatal) |

---

Next: {derive at runtime per `_preamble.md` § Lookup Procedure — intake state, default first: `/fab-continue, /fab-ff, /fab-fff, /fab-proceed, or /fab-clarify`}
