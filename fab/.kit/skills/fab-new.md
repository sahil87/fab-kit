---
name: fab-new
description: "Start a new change from a natural language description. Creates the change folder, sets it active, and generates the proposal."
---

# /fab-new <description> [--branch <name>]

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

> `fab/ is not initialized. Run /fab-init first to bootstrap the project.`

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
2. Create the subdirectory: `fab/changes/{name}/checklists/` (pre-created so downstream skills don't need a separate `mkdir`)
3. If a change folder with the same name already exists (extremely unlikely given the random component), append an additional random character to `{XXXX}` and retry

### Step 3: Set Active Change

Write the change folder name (just the name, not the full path) to `fab/current`:

```
echo "{name}" > fab/current
```

This sets the new change as the active change. Any previously active change is replaced — the user can switch back with `/fab-switch`.

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

- **If on `main` or `master`**: Auto-create a branch without prompting
  - Branch name: `{branch_prefix}{change-name}` (using `git.branch_prefix` from config, which may be empty)
  - Run `git checkout -b {branch-name}` and record in `.status.yaml`
  - This is a low-value prompt removed per SRAD scoring (high R, high A, high D — Certain grade)

- **If on a feature branch** (not main/master): **Present these options to the user (do NOT auto-select)**:
  - Show current branch name
  - Note: `wt/*` branches are worktree base branches — default to **Create new branch** instead of Adopt for these.
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
confidence:
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 5.0
last_updated: {ISO 8601 timestamp}
```

**Key points**:
- `stage` is set to `proposal`
- `proposal` progress is `active` — all other stages are `pending`
- `branch` field is only present if the user chose a branch in Step 4
- `confidence` block is initialized with defaults — Step 8 overwrites with actual counts after proposal generation
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
8. After all sections are filled, append an **`## Assumptions`** section to the artifact listing all Confident and Tentative assumptions made during generation (see Assumptions Summary Block format in `_context.md`)

### Step 7: SRAD-Based Question Selection

Apply the SRAD framework (defined in `_context.md`) to all decision points encountered during proposal generation:

1. **Evaluate each decision point** against the four SRAD dimensions (Signal Strength, Reversibility, Agent Competence, Disambiguation Type)
2. **Assign a confidence grade** (Certain, Confident, Tentative, or Unresolved)
3. **For Certain and Confident decisions**: Assume silently. Confident decisions go in the Assumptions summary.
4. **For Tentative decisions**: Assume and mark with `<!-- assumed: {description} -->` in the artifact. Include in Assumptions summary.
5. **For Unresolved decisions**: Identify the top ~3 with the highest blast radius (lowest Reversibility + lowest Agent Competence). Ask these as blocking questions. Mark remaining Unresolved as `[DEFERRED]` in the Open Questions section.

**Maximum 3 questions** — the Critical Rule (see `_context.md`) requires that Unresolved decisions with low R + low A are always asked. Beyond 3, make a best guess and mark as Tentative.

**What does NOT need SRAD evaluation**:
- Implementation details (those belong in the plan)
- Testing strategy (that belongs in tasks)
- Anything deterministically answered by config, constitution, or template rules (grade: Certain)

If SRAD evaluation finds no Unresolved decisions, skip questions entirely — generate the proposal without asking.

### Step 8: Compute Confidence Score

After generating the proposal, compute the initial confidence score:

1. Count SRAD grades across the proposal:
   - **Certain**: decisions deterministically answered by config/constitution/template rules
   - **Confident**: decisions with strong signal and one obvious interpretation
   - **Tentative**: decisions marked with `<!-- assumed: ... -->` in the artifact
   - **Unresolved**: decisions asked as questions (count only those that were asked and answered — resolved Unresolved decisions become Certain or Confident)
2. Apply the confidence formula (see `_context.md` Confidence Scoring section)
3. Write the `confidence` block to `.status.yaml`

### Step 9: Mark Proposal Complete

Once the user is satisfied with the proposal (questions answered, scope agreed):

1. Update `.status.yaml`:
   - Change `progress.proposal` from `active` to `done`
   - Write the computed `confidence` block (from Step 8)
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

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | OAuth2 over SAML | Config shows REST API stack |
| 2 | Tentative | Google + GitHub providers | Most common OSS combination |

2 assumptions made (1 confident, 1 tentative). Run /fab-clarify to review.

Next: /fab-continue or /fab-ff (fast-forward all planning)
```

### Ambiguous Description (questions needed)

```
Created fab/changes/260206-x7k2-add-oauth/
Branch: 260206-x7k2-add-oauth (created)

## Proposal: Add OAuth2 Support (Draft)

{partially filled proposal content}

Before finalizing the proposal, I need to resolve 2 unresolved decisions (SRAD: low R + low A):
1. Which OAuth providers should be supported — Google only, or also GitHub/Apple?
2. Should this replace the existing password auth or supplement it?

{user answers}

{updated proposal content}

Proposal complete.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | OAuth2 over SAML | Config shows REST API stack |

1 assumption made (1 confident, 0 tentative).

Next: /fab-continue or /fab-ff (fast-forward all planning)
```

### With `--branch`

```
/fab-new --branch feature/dev-907-oauth Add OAuth2 support

Created fab/changes/260206-x7k2-add-oauth/
Branch: feature/dev-907-oauth (adopted)

## Proposal: Add OAuth2 Support

{filled proposal content}

Proposal complete.

Next: /fab-continue or /fab-ff (fast-forward all planning)
```

### No Git Integration

```
Created fab/changes/260206-x7k2-add-oauth/

## Proposal: Add OAuth2 Support

{filled proposal content}

Proposal complete.

Next: /fab-continue or /fab-ff (fast-forward all planning)
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/config.yaml` missing | Abort with: "fab/ is not initialized. Run /fab-init first to bootstrap the project." |
| `fab/constitution.md` missing | Abort with same message as above |
| No description provided | Ask: "What change do you want to make?" |
| `fab/.kit/templates/proposal.md` missing | Abort with: "Proposal template not found at fab/.kit/templates/proposal.md — kit may be corrupted." |
| `fab/changes/{name}/` already exists | Regenerate the random component (`XXXX`) and retry |
| Git branch creation fails | Report the error, skip branch integration, continue without `branch:` in `.status.yaml` |
| `--branch` name invalid for git | Report the error, skip branch integration, continue without `branch:` in `.status.yaml` |

---

Next: `/fab-continue or /fab-ff (fast-forward all planning)`
