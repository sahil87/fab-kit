---
name: fab-new
description: "Start a new change from a natural language description. Creates the change folder and generates the intake."
---

# /fab-new <description>

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Pre-flight

1. Verify `fab/config.yaml` and `fab/constitution.md` exist
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
2. **Backlog ID** (`[a-z0-9]{4}`) — read `fab/backlog.md`, search for `\[{id}\]`. Parse any Linear ID from the line; if found, fetch per #1. Store ID for folder name.
3. **Natural language** — use as-is

### Step 1: Generate Slug

Generate a 2-6 word slug (lowercase, hyphen-joined, no articles/prepositions) from the description. If a Linear issue ID was parsed in Step 0, prefix the slug with it: e.g., `DEV-988-add-oauth`. Without a Linear ID: just the slug, e.g., `add-oauth`. This slug is passed to `changeman.sh` as the `--slug` value.

### Step 2: Gap Analysis

Check for existing mechanisms or scope concerns covering the idea. If covered: present findings, let user decide. If not: proceed.

### Step 3: Create Change

Run `lib/changeman.sh new` with appropriate flags:
- `--slug <slug>` — the slug from Step 1 (includes issue ID prefix if applicable)
- `--change-id <4char>` — only if a backlog ID was detected in Step 0 (the 4-char backlog ID becomes the change ID)
- `--log-args <description>` — the original description text

Capture the folder name from stdout. The script handles date generation, random ID generation (if no `--change-id`), collision detection, directory creation, `created_by` detection, `.status.yaml` initialization, and `stageman.sh` integration.

### Step 4: Generate `intake.md`

Follow the **Intake Generation Procedure** (`_generation.md`). Load context per `_context.md` Layer 1 and generate from `fab/.kit/templates/intake.md`.

### Step 5: SRAD-Based Question Selection

Apply SRAD (`_context.md`). No fixed question cap — SRAD scoring determines count. Zero questions for clear inputs. **Conversational mode**: when 5+ Unresolved, ask one at a time until resolved or user signals done.

---

## Output

```
{if Linear: "Fetching Linear issue DEV-988...\n"}
{if backlog: "Reading fab/backlog.md for [90g5]...\nFound: DEV-988 ...\n"}
Created fab/changes/{name}/

## Intake: {Change Name}

{intake content}

Intake complete.

{if assumptions: "## Assumptions\n\n| # | Grade | Decision | Rationale |\n..."}

Next: {per state table — activation preamble + intake state}
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Config/constitution missing | Abort: "Run /fab-setup first." |
| No description | Ask for one |
| Intake template missing | Abort: "Kit may be corrupted." |
| `changeman.sh` failure | Surface stderr output to user and stop |
| Linear ticket not found / API error | Warn, treat as natural language |
| Backlog ID not found | Abort with guidance |
| `fab/backlog.md` missing | Abort: "Use natural language or Linear ID instead." |

---

Next: `/fab-switch {name} to make it active, then /fab-continue or /fab-fff or /fab-clarify`
