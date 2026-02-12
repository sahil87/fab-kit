---
name: fab-new
description: "Start a new change from a natural language description. Creates the change folder, sets it active, and generates the brief."
---

# /fab-new <description>

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Start a new change from a natural language description. Creates the change folder, initializes the status manifest, generates the brief artifact, and calls `/fab-switch` internally to activate the change (including branch integration).

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

If no description is provided, ask the user: *"What change do you want to make?"*

---

## Behavior

### Step 1: Generate Folder Name

Generate a unique folder name using the format `{YYMMDD}-{XXXX}-{slug}`:

| Component | How to generate | Constraints |
|-----------|----------------|-------------|
| `YYMMDD` | Today's date | 6 digits, zero-padded (e.g., `260206`) |
| `XXXX` | 4 random characters | Lowercase alphanumeric only (`a-z`, `0-9`) |
| `slug` | 2-6 words extracted from description | All lowercase, words joined with `-`, no special characters |

**Slug generation rules**:
- Extract the most descriptive 2-6 words from the description
- Drop articles (a, an, the), prepositions (for, to, with, in, on, of, from), and conjunctions (and, or, but)
- Use lowercase only — avoids collisions on case-insensitive filesystems (macOS default, Windows)
- Join words with hyphens

**Examples**:
- "Add OAuth2 support for Google and GitHub sign-in" → `add-oauth`
- "Fix checkout bug in payment flow" → `fix-checkout-bug`
- "Refactor authentication middleware" → `refactor-auth-middleware`

### Step 2: Gap Analysis

Before creating the change folder, perform gap analysis to avoid redundant or overlapping changes:

1. **Check existing mechanisms** — search the workflow, codebase, and centralized docs for features or mechanisms that already cover the user's idea
2. **Evaluate scope** — is the idea too broad (should be split into multiple changes) or too narrow (part of a larger ongoing change)?
3. **Consider alternatives** — simpler approaches, extending existing skills, or configuration changes that solve the problem without new code

**If an existing mechanism covers the idea**, present findings to the user and let them decide whether to proceed:
- If the user chooses to cancel, output a brief explanation and stop — no folder created, no `Next:` line
- If the user chooses to proceed, continue to Step 3

**If no existing mechanism covers the idea**, proceed directly to Step 3.

### Step 3: Create Change Directory

1. Create the directory: `fab/changes/{name}/`
2. Create the subdirectory: `fab/changes/{name}/checklists/` (pre-created so downstream skills don't need a separate `mkdir`)
3. If a change folder with the same name already exists (extremely unlikely given the random component), regenerate the 4-character random component (`{XXXX}`) and retry

### Step 4: Initialize `.status.yaml`

Create `fab/changes/{name}/.status.yaml` using the template at `fab/.kit/templates/status.yaml`:

```yaml
name: {name}
created: {ISO 8601 timestamp}
created_by: {git config user.name, or "unknown" if unset}
progress:
  spec: active
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
- `created_by` is populated from `git config user.name`. If the command returns empty or exits non-zero, use `"unknown"` as the fallback. This field is write-once — set here and never modified by subsequent skills.
- No `stage:` field — the current stage is derived from the `active` entry in the progress map
- `spec` is `active` (first pipeline stage) — all other stages are `pending`
- `confidence` block is initialized with defaults — Step 8 overwrites with actual counts after brief generation
- Both `created` and `last_updated` use the same timestamp (current time in ISO 8601 format with timezone)

### Step 5: Generate `brief.md`

Load context before generating:
- Read `fab/config.yaml` — project name, tech stack, conventions
- Read `fab/constitution.md` — project principles and constraints
- Read `fab/docs/index.md` — understand the existing documentation landscape

Generate `fab/changes/{name}/brief.md` using the template at `fab/.kit/templates/brief.md`:

1. Read the template from `fab/.kit/templates/brief.md`
2. Fill in the metadata fields:
   - `{CHANGE_NAME}`: The human-readable description provided by the user
   - `{YYMMDD-XXXX-slug}`: The generated change folder name
   - `{DATE}`: Today's date
3. Fill in the **Origin** section — capture the user's raw input/prompt verbatim. If conversational mode was used (Step 6), also include a summary of key decisions from the conversation (not the full transcript).
4. Fill in the **Why** section — explain the motivation based on the user's description
5. Fill in the **What Changes** section — be specific about new capabilities, modifications, or removals
6. Fill in the **Affected Docs** section — identify which centralized docs (in `fab/docs/`) will be new, modified, or removed by this change. Use `fab/docs/index.md` to understand what exists.
7. Fill in the **Impact** section — identify affected code areas, APIs, dependencies
8. Fill in the **Open Questions** section (see Step 6 below)
9. After all sections are filled, append an **`## Assumptions`** section to the artifact listing all Confident and Tentative assumptions made during generation (see Assumptions Summary Block format in `_context.md`)

### Step 6: SRAD-Based Question Selection

Apply the SRAD framework (defined in `_context.md`) to all decision points encountered during brief generation:

1. **Evaluate each decision point** against the four SRAD dimensions (Signal Strength, Reversibility, Agent Competence, Disambiguation Type)
2. **Assign a confidence grade** (Certain, Confident, Tentative, or Unresolved)
3. **For Certain and Confident decisions**: Assume silently. Confident decisions go in the Assumptions summary.
4. **For Tentative decisions**: Assume and mark with `<!-- assumed: {description} -->` in the artifact. Include in Assumptions summary.
5. **For Unresolved decisions**: Ask as questions, prioritized by lowest Reversibility + lowest Agent Competence (Critical Rule).

**No fixed question cap** — SRAD scoring determines how many questions to ask. When all decisions score Confident or Certain, generate the brief in one shot with no questions. When Unresolved decisions exist, ask questions for each.

**Conversational mode**: When the user's description has low Signal Strength across many decision points (5+ Unresolved), enter conversational mode — ask questions one at a time, each building on the previous answer. The conversation continues until all Unresolved decisions are resolved or the user signals satisfaction (e.g., "good", "done", "that's enough").

**What does NOT need SRAD evaluation**:
- Implementation details (those belong in the spec or tasks)
- Testing strategy (that belongs in tasks)
- Anything deterministically answered by config, constitution, or template rules (grade: Certain)

If SRAD evaluation finds no Unresolved decisions, skip questions entirely — generate the brief without asking.

### Step 7: Compute Confidence Score

After generating the brief, compute the initial confidence score:

1. Count SRAD grades across the brief:
   - **Certain**: decisions deterministically answered by config/constitution/template rules
   - **Confident**: decisions with strong signal and one obvious interpretation
   - **Tentative**: decisions marked with `<!-- assumed: ... -->` in the artifact
   - **Unresolved**: decisions asked as questions (count only those that were asked and answered — resolved Unresolved decisions become Certain or Confident)
2. Apply the confidence formula (see `_context.md` Confidence Scoring section)
3. Write the `confidence` block to `.status.yaml`

### Step 8: Mark Brief Complete

Once the user is satisfied with the brief (questions answered, scope agreed):

1. Update `.status.yaml`:
   - Write the computed `confidence` block (from Step 7)
   - Update `last_updated` to current timestamp
2. The brief is an input artifact, not a pipeline stage — `.status.yaml` progress remains as initialized (`spec: active`)

### Step 9: Activate Change via `/fab-switch`

Invoke the `/fab-switch` flow internally to activate the change:

1. Call `/fab-switch {name}` — this writes the change name to `fab/current` and performs branch integration (if `git.enabled`)
2. From the user's perspective, `/fab-new` still results in an active change with a branch — the internal delegation is transparent
3. If `/fab-switch` branch integration fails, the change is still activated (fab/current is written) — only the branch creation is affected

---

## Output

### Clear Description (no questions needed)

```
Created fab/changes/260206-x7k2-add-oauth/
Branch: 260206-x7k2-add-oauth (created)

## Brief: Add OAuth2 Support

{filled brief content}

Brief complete.

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

## Brief: Add OAuth2 Support (Draft)

{partially filled brief content}

Before finalizing the brief, I need to resolve 2 unresolved decisions (SRAD: low R + low A):
1. Which OAuth providers should be supported — Google only, or also GitHub/Apple?
2. Should this replace the existing password auth or supplement it?

{user answers}

{updated brief content}

Brief complete.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | OAuth2 over SAML | Config shows REST API stack |

1 assumption made (1 confident, 0 tentative).

Next: /fab-continue or /fab-ff (fast-forward all planning)
```

### No Git Integration

```
Created fab/changes/260206-x7k2-add-oauth/

## Brief: Add OAuth2 Support

{filled brief content}

Brief complete.

Next: /fab-continue or /fab-ff (fast-forward all planning)
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/config.yaml` missing | Abort with: "fab/ is not initialized. Run /fab-init first to bootstrap the project." |
| `fab/constitution.md` missing | Abort with same message as above |
| No description provided | Ask: "What change do you want to make?" |
| `fab/.kit/templates/brief.md` missing | Abort with: "Brief template not found at fab/.kit/templates/brief.md — kit may be corrupted." |
| `fab/changes/{name}/` already exists | Regenerate the random component (`XXXX`) and retry |

---

Next: `/fab-continue or /fab-ff (fast-forward all planning)`
