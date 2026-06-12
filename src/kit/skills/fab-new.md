---
name: fab-new
description: "Start a new change — creates the intake, activates it, and creates the git branch."
helpers: [_generation, _srad]
---

# /fab-new <description>

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

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

### Step 0: Parse Input

Detect input type (check in order):

1. **Linear ticket ID** (`[A-Z]+-\d+`) — fetch via `mcp__claude_ai_Linear__get_issue`; extract title, description, state, labels, branchName. On failure, fall back to natural language.
2. **Backlog ID** (`[a-z0-9]{4}`) — read `fab/backlog.md`, search for `\[{id}\]`. Check for an optional `[ISSUE_ID]` bracket immediately after (e.g., `[ni3o] [DEV-1011]`); if found, extract and fetch per #1. Store backlog ID for folder name.
3. **Natural language** — use as-is

### Step 1: Generate Slug

Generate a 2-6 word slug (lowercase, hyphen-joined, no articles/prepositions) from the description. The slug SHALL NOT include the Linear issue ID — it contains only the descriptive portion (e.g., `add-oauth`). This slug is passed to `fab change new` as the `--slug` value.

### Step 2: Gap Analysis

Check for existing mechanisms or scope concerns covering the idea. If covered: present findings, let user decide. If not: proceed.

### Step 3: Create Change

**Re-run / collision check** (only when a backlog or Linear ID was detected in Step 0): before creating, check whether a non-archived change already exists for that ID. The mechanism differs by ID type:

- **Backlog ID** (4-char — embedded in the folder-name prefix): `fab change resolve {id} 2>/dev/null` — a successful resolution names the existing change.
- **Linear ID** (never in folder names — slugs exclude issue IDs; the ID is recorded only in `.status.yaml` `issues` arrays): `grep -lw "{ISSUE_ID}" fab/changes/*/.status.yaml 2>/dev/null` — `-w` anchors on word boundaries so `DEV-123` does not match `DEV-1234`; the single-level glob naturally excludes `fab/changes/archive/`; a match's parent folder is the existing change.

If a check finds an existing change, do NOT create a duplicate — **route to resume**: report `Change {name} already exists for [{id}].` and point the user to `/fab-switch {name}` then `/fab-continue` (whose intake-`active` dispatch row regenerates a missing intake, recovering an interrupted creation). STOP. (For backlog IDs, `fab change new`'s `Change ID already in use` error remains the safety net if this check is skipped; Linear re-runs have no CLI safety net — no `--change-id` is passed — so this scan is the only collision guard.)

**Natural-language re-run semantics**: a natural-language description intentionally creates a **new change on every run** (fresh random ID) — there is no dedup for NL input.

Run `fab change new` with appropriate flags:
- `--slug <slug>` — the slug from Step 1 (descriptive only, no issue ID)
- `--change-id <4char>` — only if a backlog ID was detected in Step 0 (the 4-char backlog ID becomes the change ID)
- `--log-args <description>` — the original description text

Capture the folder name from stdout. The command handles date generation, random ID generation (if no `--change-id`), collision detection, directory creation, `created_by` detection, `.status.yaml` initialization, and command logging (when `--log-args` is provided).

If a Linear ticket was detected in Step 0, record the issue ID via `fab status`:
`fab status add-issue {name} DEV-988` (using the actual detected ID).

### Step 4: Conversation Context Mining

Before generating the intake, scan the current conversation for prior discussion of this change's topic — whether from `/fab-discuss`, free-form exploration, or any conversation that preceded this `/fab-new` invocation. Extract:

- **Decisions made** — specific choices with rationale (e.g., "OAuth2 over SAML because no enterprise requirement")
- **Alternatives rejected** — options considered and why they were ruled out
- **Constraints identified** — boundaries or requirements surfaced during discussion
- **Specific values agreed upon** — config structures, API shapes, exact behaviors

Encode extracted decisions as Certain or Confident assumptions in the intake's Assumptions table with rationale referencing the discussion (e.g., "Discussed — user chose X over Y"). These feed directly into SRAD scoring and reduce downstream ambiguity.

If no prior discussion exists in the conversation, skip this step — behavior is identical to a cold `/fab-new`.

### Step 5: Generate `intake.md`

Follow the **Intake Generation Procedure** (`_generation.md`). Load context per `_preamble.md` Layer 1 and generate from `$(fab kit-path)/templates/intake.md`. Incorporate any decisions extracted in Step 4.

### Step 6: Verify Change Type

The PostToolUse intake-write hook owns `change_type`: it infers and writes the type to `.status.yaml` on **every** `intake.md` write, using word-boundary keyword regexes evaluated in order — `fix` → `refactor` (incl. "redesign") → `docs` → `test` → `ci` → `chore` — defaulting to `feat`. Do NOT run a manual keyword inference or an unconditional `set-change-type`: any later intake write (e.g., `/fab-clarify`) re-fires the hook and silently overwrites a skill-set value.

1. **Verify** the hook's result by reading `change_type` from the change's `.status.yaml` (e.g., `grep '^change_type:' fab/changes/{name}/.status.yaml`) — `fab preflight` does not emit this field
2. **Override only if wrong**: `fab status set-change-type {name} <type>` — and note that any subsequent intake edit re-fires the hook and overwrites the override, so re-verify after later intake writes

### Step 7: Confidence

After generating `intake.md` and verifying the change type, persist and display the confidence score:

1. Call `fab score --stage intake <change>` (normal mode, **not** `--check-gate`)
2. This writes the score to `.status.yaml` (no `indicative` flag is written — retired in 1.10.0; intake scoring is authoritative)
3. Display the result from stdout (score and breakdown)

Output format: `Confidence: {score} / 5.0 ({N} decisions)`

The score is persisted to `.status.yaml` so that consumers (`/fab-switch`, `/fab-status`, `fab change list`) can display it without recomputation. It is the authoritative confidence — intake is the sole scoring source, and the single intake gate (flat 3.0) reads it.

### Step 8: SRAD-Based Question Selection

Apply SRAD (`_srad.md`, loaded via `helpers:`). No fixed question cap — SRAD scoring determines count. Zero questions for clear inputs. **Conversational mode**: when 5+ Unresolved, ask one at a time until resolved or user signals done.

### Step 9: Advance Intake to Ready

After all intake work is complete (generation, type verification, confidence, questions), advance intake to `ready`:

```bash
fab status advance {name} intake
```

This signals that the intake artifact exists and is open for `/fab-clarify` refinement. After Step 10 activates the change, the user can run `/fab-continue` immediately to proceed to apply (which co-generates `plan.md` — requirements + tasks + acceptance).

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
fab change resolve "$(git branch --show-current)" 2>/dev/null                              # which change (if any) does the current branch belong to?
```

> **fab-new-specific `{dirty_count}` derivation**: the porcelain count excludes `fab/changes/{name}/` — this change's own just-created artifacts (`intake.md`, `.status.yaml`, `.history.jsonl`) always exist uncommitted by Step 11, so counting them would fire the dirty-tree note on every run. Only *pre-existing* uncommitted work should trigger the note.

<!-- Keep this table in sync with git-branch.md Step 4 — same cases, same commands, same report strings (incl. the rename guard, the remote-only --track case, and the dirty-tree note). One deliberate divergence lives OUTSIDE the shared rows: fab-new derives {dirty_count} excluding fab/changes/{name}/ (see the derivation note above); git-branch counts the full porcelain output. -->

**Evaluate in order, first match wins:**

| # | Condition | Command | Report |
|---|-----------|---------|--------|
| 1 | Current branch equals `{name}` | *(none)* | `Branch: {name} (already active)` |
| 2 | Target branch `{name}` exists locally (`git rev-parse --verify "{name}"` succeeds) | `git checkout "{name}"` | `Branch: {name} (checked out)` |
| 3 | Target exists only on the remote (`origin/{name}` verify succeeds) — do NOT recreate a divergent local | `git checkout --track "origin/{name}"` | `Branch: {name} (checked out, tracking origin/{name})` |
| 4 | On `main` or `master` | `git checkout -b "{name}"` | `Branch: {name} (created)` |
| 5 | Local-only branch (`upstream` empty) AND the **rename guard** passes: the current branch belongs to no change (`fab change resolve` fails — e.g., a disposable `wt create` name) or resolves to this SAME change (e.g., a worktree placeholder named with the change's own ID) | `git branch -m "{name}"` | `Branch: {name} (renamed from {old_branch})` |
| 6 | Local-only branch belonging to a different change (`fab change resolve` succeeds with another change — e.g., after `/fab-switch`; do NOT rename it away, caveat: the new branch inherits the old change's HEAD) OR pushed branch (`upstream` non-empty) | `git checkout -b "{name}"` | `Branch: {name} (created, leaving {old_branch} intact)` |

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

Next: `/fab-continue, /fab-fff, /fab-ff, or /fab-clarify`
