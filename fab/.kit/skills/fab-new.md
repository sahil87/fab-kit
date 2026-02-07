---
name: fab-new
description: "Start a new change from a natural language description. Creates the change folder, sets it active, and generates the proposal."
---

# /fab:new <description> [--branch <name>]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Start a new change from a natural language description. Creates the change folder, sets it as the active change, optionally integrates with git, initializes the status manifest, and generates the proposal artifact.

---

## Pre-flight Check

Before doing anything else:

1. Check that `fab/config.yaml` exists and is readable
2. Check that `fab/constitution.md` exists and is readable

**If either check fails, STOP immediately.** Output this message and do nothing else:

> `fab/ is not initialized. Run /fab:init first to bootstrap the project.`

Do NOT create partial structure. The project must be initialized before starting a new change.

---

## Arguments

- **`<description>`** *(required)* — natural language description of the change (e.g., "Add OAuth2 support for Google and GitHub sign-in")
- **`--branch <name>`** *(optional)* — explicit branch name to use. Skips the branch prompt and uses this name directly. Useful for Linear-linked branches, team conventions, or pre-existing branches.

If no description is provided, ask the user: *"What change do you want to make?"*

---

## Behavior

### Step 1: Generate Folder Name

Generate a unique folder name using the format `{YYMMDD}-{XXXX}-{slug}`:

| Component | How to generate | Constraints |
|-----------|----------------|-------------|
| `YYMMDD` | Today's date | 6 digits, zero-padded (e.g., `260206`) |
| `XXXX` | 4 random characters | Lowercase alphanumeric only (`a-z`, `0-9`) |
| `slug` | 2-4 words extracted from description | All lowercase, words joined with `-`, no special characters |

**Slug generation rules**:
- Extract the most descriptive 2-4 words from the description
- Drop articles (a, an, the), prepositions (for, to, with, in, on, of, from), and conjunctions (and, or, but)
- Use lowercase only — avoids collisions on case-insensitive filesystems (macOS default, Windows)
- Join words with hyphens

**Examples**:
- "Add OAuth2 support for Google and GitHub sign-in" → `add-oauth`
- "Fix checkout bug in payment flow" → `fix-checkout-bug`
- "Refactor authentication middleware" → `refactor-auth-middleware`

### Step 2: Create Change Directory

1. Create the directory: `fab/changes/{name}/`
2. If a change folder with the same name already exists (extremely unlikely given the random component), append an additional random character to `{XXXX}` and retry

### Step 3: Set Active Change

Write the change folder name (just the name, not the full path) to `fab/current`:

```
echo "{name}" > fab/current
```

This sets the new change as the active change. Any previously active change is replaced — the user can switch back with `/fab:switch`.

### Step 4: Git Integration

**Skip this step entirely if**:
- `git.enabled` is `false` in `fab/config.yaml`, OR
- The project is not inside a git repository

**If `--branch <name>` was provided**:
1. Use the provided name directly
2. If a git branch with that name already exists, adopt it (check it out if not already on it)
3. If it doesn't exist, create it: `git checkout -b <name>`
4. Record the branch name in `.status.yaml` as `branch: <name>`
5. Skip the interactive prompt below

**If no `--branch` argument** (interactive flow):

Detect the current branch and offer options:

- **If on `main` or `master`**: Offer to create a new branch
  - Suggest branch name: `{branch_prefix}{change-name}` (using `git.branch_prefix` from config, which may be empty)
  - Options: **Create branch** (default), **Skip** (no branch tracking)
  - If user chooses "Create branch": `git checkout -b {branch-name}` and record in `.status.yaml`
  - If user chooses "Skip": no `branch:` field in `.status.yaml`

- **If on a feature branch** (not main/master): Offer to adopt it
  - Show current branch name
  - Options: **Adopt this branch** (default), **Create new branch**, **Skip**
  - If user chooses "Adopt": record the current branch name as-is in `.status.yaml`
  - If user chooses "Create new branch": create a new branch as above
  - If user chooses "Skip": no `branch:` field in `.status.yaml`

### Step 5: Initialize `.status.yaml`

Create `fab/changes/{name}/.status.yaml` with this content:

```yaml
name: {name}
created: {ISO 8601 timestamp}
branch: {branch-name}  # Only include if branch was set in Step 4
stage: proposal
progress:
  proposal: active
  specs: pending
  plan: pending
  tasks: pending
  apply: pending
  review: pending
  archive: pending
checklist:
  generated: false
  path: checklists/quality.md
  completed: 0
  total: 0
last_updated: {ISO 8601 timestamp}
```

**Key points**:
- `stage` is set to `proposal`
- `proposal` progress is `active` — all other stages are `pending`
- `branch` field is only present if the user chose a branch in Step 4
- Both `created` and `last_updated` use the same timestamp (current time in ISO 8601 format with timezone)

### Step 6: Generate `proposal.md`

Load context before generating:
- Read `fab/config.yaml` — project name, tech stack, conventions
- Read `fab/constitution.md` — project principles and constraints
- Read `fab/docs/index.md` — understand the existing documentation landscape

Generate `fab/changes/{name}/proposal.md` using the template at `fab/.kit/templates/proposal.md`:

1. Read the template from `fab/.kit/templates/proposal.md`
2. Fill in the metadata fields:
   - `{CHANGE_NAME}`: The human-readable description provided by the user
   - `{YYMMDD-XXXX-slug}`: The generated change folder name
   - `{DATE}`: Today's date
3. Fill in the **Why** section — explain the motivation based on the user's description
4. Fill in the **What Changes** section — be specific about new capabilities, modifications, or removals
5. Fill in the **Affected Docs** section — identify which centralized docs (in `fab/docs/`) will be new, modified, or removed by this change. Use `fab/docs/index.md` to understand what exists.
6. Fill in the **Impact** section — identify affected code areas, APIs, dependencies
7. Fill in the **Open Questions** section (see Step 7 below)

### Step 7: Clarifying Questions

If the user's description is ambiguous or leaves significant questions unanswered:

1. Identify up to 3 questions that **must** be resolved before specs can be written
2. Mark these as `[BLOCKING]` in the proposal's Open Questions section
3. Ask the user these questions directly
4. **Maximum 3 blocking questions** — for anything else, make an informed guess and note it as `[DEFERRED]`

**What counts as ambiguous**:
- Scope unclear (e.g., "improve auth" — which aspect?)
- Multiple valid approaches with different tradeoffs
- Missing technical context the agent can't infer from the codebase
- Conflicting signals in the description

**What does NOT need clarification**:
- Implementation details (those belong in the plan)
- Testing strategy (that belongs in tasks)
- Anything the agent can reasonably infer from config, constitution, or existing docs

If the description is clear and unambiguous, skip this step — generate the proposal without questions.

### Step 8: Mark Proposal Complete

Once the user is satisfied with the proposal (questions answered, scope agreed):

1. Update `.status.yaml`:
   - Change `progress.proposal` from `active` to `done`
   - Update `last_updated` to current timestamp
2. The proposal status field in the proposal.md itself can remain as-is (the `.status.yaml` is the source of truth)

---

## Output

### Clear Description (no questions needed)

```
Created fab/changes/260206-x7k2-add-oauth/
Branch: 260206-x7k2-add-oauth (created)

## Proposal: Add OAuth2 Support

{filled proposal content}

Proposal complete.

Next: /fab:continue or /fab:ff (fast-forward all planning)
```

### Ambiguous Description (questions needed)

```
Created fab/changes/260206-x7k2-add-oauth/
Branch: 260206-x7k2-add-oauth (created)

## Proposal: Add OAuth2 Support (Draft)

{partially filled proposal content}

Before finalizing the proposal, I need to clarify:
1. [BLOCKING] Which OAuth providers should be supported — Google only, or also GitHub/Apple?
2. [BLOCKING] Should this replace the existing password auth or supplement it?

{user answers}

{updated proposal content}

Proposal complete.

Next: /fab:continue or /fab:ff (fast-forward all planning)
```

### With `--branch`

```
/fab:new --branch feature/dev-907-oauth Add OAuth2 support

Created fab/changes/260206-x7k2-add-oauth/
Branch: feature/dev-907-oauth (adopted)

## Proposal: Add OAuth2 Support

{filled proposal content}

Proposal complete.

Next: /fab:continue or /fab:ff (fast-forward all planning)
```

### No Git Integration

```
Created fab/changes/260206-x7k2-add-oauth/

## Proposal: Add OAuth2 Support

{filled proposal content}

Proposal complete.

Next: /fab:continue or /fab:ff (fast-forward all planning)
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/config.yaml` missing | Abort with: "fab/ is not initialized. Run /fab:init first to bootstrap the project." |
| `fab/constitution.md` missing | Abort with same message as above |
| No description provided | Ask: "What change do you want to make?" |
| `fab/.kit/templates/proposal.md` missing | Abort with: "Proposal template not found at fab/.kit/templates/proposal.md — kit may be corrupted." |
| `fab/changes/{name}/` already exists | Regenerate the random component (`XXXX`) and retry |
| Git branch creation fails | Report the error, skip branch integration, continue without `branch:` in `.status.yaml` |
| `--branch` name invalid for git | Report the error, skip branch integration, continue without `branch:` in `.status.yaml` |

---

Next: `/fab:continue or /fab:ff (fast-forward all planning)`
