---
name: _intake
description: "Shared pre-boundary Create-Intake Procedure (fab-new Steps 0–9) used by fab-new, fab-draft, and fab-proceed — parse input, slug, gap analysis, create change, conversation context mining, generate intake.md, verify change type, confidence, question selection, advance to ready. Parameterized by a single {questioning-mode} knob (interactive | promptless-defer)."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Shared Create-Intake Procedure

> This file defines the shared **pre-boundary** intake-creation logic used by three skills:
> `/fab-new`, `/fab-draft`, and `/fab-proceed` (its fab-new subagent dispatch). The calling skill
> (the **consumer**) binds one parameter before executing this procedure — read it from the
> consumer's own file:
>
> - **`{questioning-mode}`** — how Step 8 resolves ambiguity:
>   - **`interactive`** — used by `/fab-new` and `/fab-draft`. Step 8 asks the user via SRAD
>     (SRAD-driven question selection, no fixed cap, conversational mode when 5+ Unresolved).
>   - **`promptless-defer`** — used by `/fab-proceed`'s dispatch. Step 8 records each would-be-asked
>     Unresolved decision as a deferred Unresolved row instead of asking, per the `_srad.md`
>     promptless-dispatch carve-out (quoted at Step 8).
>
> This is the **only** behavioral fork in intake creation, and it is legitimately
> **invocation-level** (who resolves ambiguity: human-now vs. defer-and-surface). It is exactly
> parallel to the post-boundary autonomy fork (interactive rework menu vs. autonomous auto-rework).
>
> **What stays at the call site** (do NOT do it here): deciding *whether* to create an intake
> (`/fab-proceed`'s state-detection + relevance assessment), and what to do *after* (activate +
> git branch — `/fab-new`'s Steps 10–11 tail). This procedure is purely "given I've decided to
> create an intake, do it (Steps 0–9)."
>
> This procedure references `_generation.md` (Step 5) and `_srad.md` (Steps 4, 8) in-body; it carries
> no `helpers:` frontmatter of its own — every consumer already declares both helpers (the
> consumer-declared model), or, for `/fab-proceed`, dispatches this procedure to a subagent that loads
> them. Mirror the proven `_pipeline.md` shape: shared body parameterized by one knob, call-site
> tails stay in the call-site files.

---

## Create-Intake Procedure (Steps 0–9)

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

- **Backlog ID** (4-char — embedded in the folder-name prefix): `fab resolve --id {id} 2>/dev/null` — then compare its stdout (the canonical 4-char ID of the matched change) for **equality** with `{id}`. Only an exact ID match names an existing change for this ID; resolution is substring-based, so `{id}` occurring inside another change's *slug* also resolves — with a different canonical ID — and MUST NOT route to resume. On an exact match, get the folder name via `fab resolve --folder {id}`.
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

This step is the load-bearing **context-flush at the boundary**: intake is the single context-bearing boundary in the pipeline (everything after it runs as a dispatched subagent reading only the intake artifact). Capturing the live conversation's decisions here is what transfers that context across the boundary — it is not merely a scoring optimization.

Before generating the intake, scan the current conversation for prior discussion of this change's topic — whether from `/fab-discuss`, free-form exploration, or any conversation that preceded the invoking skill's invocation. Extract:

- **Decisions made** — specific choices with rationale (e.g., "OAuth2 over SAML because no enterprise requirement")
- **Alternatives rejected** — options considered and why they were ruled out
- **Constraints identified** — boundaries or requirements surfaced during discussion
- **Specific values agreed upon** — config structures, API shapes, exact behaviors

Encode extracted decisions as Certain or Confident assumptions in the intake's Assumptions table with rationale referencing the discussion (e.g., "Discussed — user chose X over Y"). These feed directly into SRAD scoring and reduce downstream ambiguity.

If no prior discussion exists in the conversation, skip this step — behavior is identical to a cold invocation.

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

### Step 8: SRAD-Based Question Selection *(the parameterized step)*

This is the sole step that varies by `{questioning-mode}`.

- **`{questioning-mode} = interactive`** (used by `/fab-new`, `/fab-draft`): Apply SRAD (`_srad.md`). No fixed question cap — SRAD scoring determines count. Zero questions for clear inputs. **Conversational mode**: when 5+ Unresolved, ask one at a time until resolved or user signals done.

- **`{questioning-mode} = promptless-defer`** (used by `/fab-proceed`'s dispatch): there is no user to ask. Apply SRAD, but instead of asking, record each would-be-asked Unresolved decision as a deferred Unresolved row, per the `_srad.md` § Critical Rule **promptless-dispatch carve-out** (verbatim):

  > each would-be-asked Unresolved decision is recorded as an Unresolved row with Rationale `Deferred — promptless dispatch` and surfaced to the user by the dispatcher. The intake gate is the structural backstop — `fab score` returns 0.0 whenever any Unresolved row exists, so a deferred decision always blocks the automated bracket until resolved via `/fab-clarify`.

  Return the deferred Unresolved decisions in the subagent result so the dispatcher (`/fab-proceed`) can surface them. The MUST-ask is satisfied by deferring and surfacing, never by silently assuming.

### Step 9: Advance Intake to Ready

After all intake work is complete (generation, type verification, confidence, questions), advance intake to `ready`:

```bash
fab status advance {name} intake
```

This signals that the intake artifact exists and is open for `/fab-clarify` refinement. What happens next is the **call site's** responsibility (activate + branch, or stop at ready, or hand the deferred decisions to the dispatcher) — see the consumer's own tail.
