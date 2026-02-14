---
name: fab-new
description: "Start a new change from a natural language description. Creates the change folder, sets it active, and generates the brief."
---

# /fab-new <description> [--switch]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Pre-flight

1. Verify `fab/config.yaml` and `fab/constitution.md` exist
2. **If either missing, STOP**: `fab/ is not initialized. Run /fab-init first to bootstrap the project.`

---

## Arguments

- **`<description>`** *(required)* — natural language, Linear ticket ID (`DEV-988`), or backlog ID (`90g5`)
- **`--switch`** *(optional)* — activate after creation (calls `/fab-switch` internally). Also triggered by intent: "and switch to it", "make it active", "activate it"

If no description: ask *"What change do you want to make?"*

---

## Behavior

### Step 0: Parse Input

Detect input type (check in order):

1. **Linear ticket ID** (`[A-Z]+-\d+`) — fetch via `mcp__claude_ai_Linear__get_issue`; extract title, description, state, labels, branchName. On failure, fall back to natural language.
2. **Backlog ID** (`[a-z0-9]{4}`) — read `fab/backlog.md`, search for `\[{id}\]`. Parse any Linear ID from the line; if found, fetch per #1. Store ID for folder name.
3. **Natural language** — use as-is

### Step 1: Generate Folder Name

Format: `{YYMMDD}-{XXXX}-{slug}` — date (6 digits), backlog ID or 4 random `[a-z0-9]`, 2-6 word slug (lowercase, hyphen-joined, no articles/prepositions).

### Step 2: Gap Analysis

Check for existing mechanisms or scope concerns covering the idea. If covered: present findings, let user decide. If not: proceed.

### Step 3: Create Change Directory

Create `fab/changes/{name}/`. Backlog ID collision → abort with redirect to existing change. Random ID collision → regenerate and retry.

### Step 4: Initialize `.status.yaml`

Create from `fab/.kit/templates/status.yaml`. Fill `{NAME}` (folder name), `{CREATED}` (ISO 8601 with tz), `{CREATED_BY}` (`gh api user --jq .login` → `git config user.name` → `"unknown"`, silent fallback).

After creation, run `lib/stageman.sh set-state <file> brief active fab-new` to activate brief stage with metrics tracking, then run `lib/stageman.sh log-command <change_dir> "fab-new" "<description>"`.

### Step 5: Generate `brief.md`

Load context per `_context.md` Layer 1. Generate from `fab/.kit/templates/brief.md`:

1. Fill metadata: `{CHANGE_NAME}`, `{YYMMDD-XXXX-slug}`, `{DATE}`
2. **Origin**: Raw input + Linear/backlog ID if applicable; key decisions from Step 6 if conversational
3. **Why**: Linear description or motivation from description
4. **What Changes**: Capabilities added/modified/removed; Linear acceptance criteria if present
5. **Affected Memory**: Which `docs/memory/` files will be new/modified/removed
6. **Impact**: Affected code areas, APIs, dependencies
7. **Open Questions**: From Step 6
8. Append `## Assumptions` section (see `_context.md`)

### Step 6: SRAD-Based Question Selection

Apply SRAD (`_context.md`). No fixed question cap — SRAD scoring determines count. Zero questions for clear inputs. **Conversational mode**: when 5+ Unresolved, ask one at a time until resolved or user signals done.

### Step 7: Activate Change (Conditional)

Default: skip. Switch if `--switch` or intent detected. Calls `/fab-switch {name}` transparently.

---

## Output

```
{if Linear: "Fetching Linear issue DEV-988...\n"}
{if backlog: "Reading fab/backlog.md for [90g5]...\nFound: DEV-988 ...\n"}
Created fab/changes/{name}/
{if switched: "Branch: {name} (created)\n"}

## Brief: {Change Name}

{brief content}

Brief complete.

{if assumptions: "## Assumptions\n\n| # | Grade | Decision | Rationale |\n..."}

Next: {per _context.md Next Steps table}
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Config/constitution missing | Abort: "Run /fab-init first." |
| No description | Ask for one |
| Brief template missing | Abort: "Kit may be corrupted." |
| Backlog ID collision | Abort: redirect to existing change |
| Random ID collision | Regenerate and retry |
| Linear ticket not found / API error | Warn, treat as natural language |
| Backlog ID not found | Abort with guidance |
| `fab/backlog.md` missing | Abort: "Use natural language or Linear ID instead." |

---

Next (default): `/fab-switch {name} to make it active, then /fab-continue or /fab-ff`

Next (with `--switch`): `/fab-continue or /fab-ff`
