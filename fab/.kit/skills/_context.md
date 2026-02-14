# Shared Context Preamble

> This file defines shared conventions for all Fab skills. Each skill file should begin with:
> `Read and follow the instructions in fab/.kit/skills/_context.md before proceeding.`

---

## Context Loading

Before generating or validating any artifact, load the relevant context layers below. This ensures output is grounded in the actual project state, not assumptions.

### 1. Always Load (every skill except `/fab-init`, `/fab-status`, `/docs-hydrate-memory`; `/fab-switch` loads only `config.yaml`)

Read these files first — they define the project's identity, constraints, and documentation landscape:

- **`fab/config.yaml`** — project configuration, tech stack, naming conventions, stage configuration
- **`fab/constitution.md`** — project principles and constraints (MUST/SHOULD/MUST NOT rules)
- **`fab/memory/index.md`** — memory landscape (which domains and memory files exist)
- **`fab/specs/index.md`** — specifications landscape (pre-implementation design intent, human-curated)

> **Note**: If the skill runs `fab-preflight.sh` (Section 2 above), the init check (config.yaml and constitution.md existence) is already covered by the script. Skills using preflight don't need separate existence checks for these files — they only need to read them for content.

### 2. Change Context (when operating on an active change)

Resolve the active change and load its state by running the preflight script:

1. **Run preflight**: Execute `fab/.kit/scripts/fab-preflight.sh [change-name]` via Bash — pass the optional change-name argument if the skill received one
2. **Check exit code**: If the script exits non-zero, STOP and surface the stderr message to the user (it contains the specific error and suggested fix)
3. **Parse stdout YAML**: On success, parse the YAML output for `name`, `change_dir`, `stage`, `progress`, `checklist`, and `confidence` fields — use these for all subsequent change context instead of re-reading `.status.yaml`
4. Load all completed artifacts in the change folder (e.g., `brief.md`, `spec.md`, `tasks.md`) — read each file that exists so you have full context of what has been decided so far

> **Change-name override**: When a `[change-name]` argument is passed to the preflight script, it resolves the change using case-insensitive substring matching against `fab/changes/` folder names (excluding `archive/`) instead of reading `fab/current`. The override is **transient** — `fab/current` is never modified. This enables parallel workflows where multiple tabs target different changes concurrently. Supports full folder names, partial slugs, or 4-char IDs (e.g., `r3m7`).

> **What the script validates internally** (for reference — agents do not need to duplicate these checks):
> 1. `fab/config.yaml` and `fab/constitution.md` exist (project initialized)
> 2. `fab/current` exists and is non-empty (active change set) — OR `$1` override resolves to a valid change
> 3. Change directory `fab/changes/{name}/` exists
> 4. `.status.yaml` exists within the change directory

### 3. Memory File Lookup (when operating on an active change)

Selectively load relevant memory files based on the change's scope:

1. Read the brief's **Affected Memory** section (or spec's **Affected memory** metadata) to identify which domains are relevant
2. For each referenced domain, read `fab/memory/{domain}/index.md` to understand the domain's memory files
3. Read the specific memory file(s) referenced by the Affected Memory entries (those marked `(new)`, `(modify)`, or `(remove)`) — read `fab/memory/{domain}/{name}.md` for each listed file that exists
4. If a referenced file or domain does not exist yet (e.g., listed as `(new)`), note this and proceed without error — it will be created during hydrate (via `/fab-continue` or `/fab-ff`)
5. Use this context to ground all artifact generation (spec, tasks, reviews) in the real current state, not assumptions

### 4. Source Code Loading (during implementation and review)

Load only the source files relevant to the current work:

1. Read the relevant source files referenced in the task descriptions or spec's affected areas
2. Scope to files actually touched by the change — do not load the entire codebase
3. This applies primarily to apply and review behavior in `/fab-continue`

---

## Next Steps Convention

Every skill MUST end its output with a `Next:` line suggesting the available follow-up commands. This keeps the user oriented in the workflow without needing to memorize the stage graph.

**Format**: `Next: /fab-command` or `Next: /fab-commandA or /fab-commandB (description)`

### Lookup Table

| After skill | Stage reached | Next line |
|-------------|---------------|-----------|
| `/fab-init` | initialized | `Next: /fab-new <description> or /docs-hydrate-memory <sources>` |
| `/docs-hydrate-memory` | memory hydrated | `Next: /fab-new <description> or /docs-hydrate-memory <more-sources>` |
| `/fab-new` | brief active | `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-ff` |
| `/fab-continue` → spec | spec done | `Next: /fab-continue or /fab-ff or /fab-clarify` |
| `/fab-continue` → tasks | tasks done | `Next: /fab-continue or /fab-ff` |
| `/fab-continue` → apply | apply done | `Next: /fab-continue` |
| `/fab-continue` → review (pass) | review done | `Next: /fab-continue` |
| `/fab-continue` → review (fail) | review failed | *(contextual rework options)* |
| `/fab-continue` → hydrate | hydrated | `Next: /fab-archive` |
| `/fab-ff` | hydrated | `Next: /fab-archive` |
| `/fab-ff` (bail) | varies | *(contextual — see /fab-ff for bail/failure messages)* |
| `/fab-clarify` | same stage | `Next: /fab-clarify or /fab-continue or /fab-ff` |
| `/fab-fff` | hydrated | `Next: /fab-archive` |
| `/fab-fff` (bail) | varies | *(contextual — see /fab-fff for bail messages)* |
| `/fab-archive` | archived | `Next: /fab-new <description>` |

---

## Skill Invocation Protocol

When one skill invokes another internally (e.g., `/fab-ff` invoking `/fab-clarify` between stages), the calling skill MUST signal the invocation mode explicitly using an instruction prefix. This makes the contract between skills explicit and testable rather than relying on implicit "call context" interpretation.

### Protocol

1. **Prefix**: `[AUTO-MODE]`
2. **Placement**: The calling skill includes `[AUTO-MODE]` as the **first line** of the invocation prompt / instruction to the called skill.
3. **Detection**: The called skill checks for the `[AUTO-MODE]` prefix at the start of its invocation context.
   - **If present**: Enter autonomous mode (no user interaction, machine-readable result).
   - **If absent**: Enter default/interactive mode (user-facing, structured questions).
4. **Transitivity**: When skills chain (e.g., `/fab-fff` → `/fab-ff` → `/fab-clarify`), each link in the chain applies the prefix independently. `/fab-ff` adds `[AUTO-MODE]` when it invokes `/fab-clarify`, regardless of whether `/fab-ff` itself was invoked by `/fab-fff` or by the user.

### Currently Applicable

| Calling skill | Called skill | Mode signaled |
|---------------|-------------|---------------|
| `/fab-ff` | `/fab-clarify` | `[AUTO-MODE]` → auto mode (autonomous gap resolution) |
| `/fab-fff` (via `/fab-ff`) | `/fab-clarify` | `[AUTO-MODE]` → auto mode (transitive through `/fab-ff`) |

User-invoked skills never carry the `[AUTO-MODE]` prefix, so called skills default to interactive mode.

To add new mode signals, define new bracketed prefixes (e.g., `[BATCH-MODE]`) here. Pattern: one prefix per mode, first-line placement, absence means default.

---

## SRAD Autonomy Framework

When generating artifacts, planning skills encounter decision points not explicitly addressed by user input. The SRAD framework provides a principled method for deciding when to ask, when to assume, and when to surface assumptions.

### SRAD Scoring

For each decision point, evaluate four dimensions:

| Dimension | High (safe to assume) | Low (consider asking) |
|-----------|----------------------|----------------------|
| **S — Signal Strength** | Detailed description, multiple sentences, clear intent | One-liner, vague phrase, ambiguous scope |
| **R — Reversibility** | Easily changed later via `/fab-clarify` or stage reset | Cascades through multiple artifacts, expensive to undo |
| **A — Agent Competence** | Config, constitution, codebase give clear answer | Business priorities, user preferences, political context |
| **D — Disambiguation Type** | One obvious default interpretation | Multiple valid interpretations with different tradeoffs |

### Confidence Grades

Each decision produces an assumption graded on a 4-level scale:

| Grade | Meaning | Artifact Marker | Output Visibility |
|-------|---------|----------------|-------------------|
| **Certain** | Determined by config/constitution/template rules | None | None — not worth mentioning |
| **Confident** | Strong signal, one obvious interpretation | None | Noted in Assumptions summary |
| **Tentative** | Reasonable guess, multiple valid options | `<!-- assumed: {description} -->` | Noted in Assumptions summary, `/fab-clarify` suggested |
| **Unresolved** | Cannot determine, incompatible interpretations | None — always asked or bailed | Asked as question (fab-new/continue), batched upfront (fab-ff/fab-fff) |

### Critical Rule

**Unresolved decisions with low Reversibility AND low Agent Competence MUST always be asked** — even in `/fab-new` and `/fab-continue`. These count toward the skill's question budget (max ~3). The existence of `/fab-clarify` as an escape valve does NOT justify silently assuming high-blast-radius decisions. `/fab-clarify` is for Tentative assumptions, not for Unresolved ones.

### Skill-Specific Autonomy Levels

| Aspect | fab-new (adaptive) | fab-continue (deliberate) | fab-ff (speed) | fab-fff (full pipeline) |
|--------|-------------------|---------------------------|----------------|-------------------------|
| **Posture** | SRAD-driven: 0 questions for clear inputs, conversational for vague; gap analysis before folder creation | Surface tentative, ask top ~3 unresolved | Batch all unresolved upfront, then go | Same as fab-ff; gated on confidence >= 3.0 |
| **Interruption budget** | SRAD-driven (no fixed cap); conversational mode for vague inputs | 1-2 per stage | 0-1 batch at start | Same as fab-ff (frontloaded) |
| **Output** | Assumptions summary + "Run /fab-clarify to review" | Key Decisions block + Assumptions summary + [NEEDS CLARIFICATION] count | Cumulative Assumptions summary | Same as fab-ff + apply/review/hydrate output |
| **Escape valve** | `/fab-clarify` | `/fab-clarify` | `/fab-clarify` | `/fab-clarify` (bails on blockers or review failure) |
| **Recomputes confidence?** | No | Spec stage only | No | No |

### Worked Examples

#### Example 1: Auth provider selection

> **Decision point**: User says "Add auth." Which provider — OAuth2, SAML, API keys?
>
> | Dimension | Score | Reasoning |
> |-----------|-------|-----------|
> | S — Signal | Low | One word ("auth") — no detail on mechanism |
> | R — Reversibility | Low | Auth architecture cascades into DB schema, middleware, API contracts |
> | A — Agent Competence | Low | Business relationship with identity providers is a user preference |
> | D — Disambiguation | Low | OAuth2, SAML, and API keys all valid with different tradeoffs |
>
> **Grade: Unresolved** — all four dimensions score low. This MUST be asked (Critical Rule applies: low R + low A).

#### Example 2: Error response format

> "Handle errors" in a REST API → S: Medium, R/A/D: High. **Confident** — codebase signal is strong, easily reversed, one obvious default. Note in Assumptions summary, don't ask.

#### Example 3: Test framework selection

> "Which test framework?" → S: Low, R/A/D: High. **Certain** — config deterministically answers this (use existing runner). No marker, no mention.

### Artifact Markers

Planning skills use HTML comment markers to flag assumptions for downstream scanning by `/fab-clarify`:

| Marker | Grade | Placed by | Scanned by |
|--------|-------|-----------|------------|
| `<!-- assumed: {description} -->` | Tentative | All planning skills (fab-new, fab-continue, fab-ff) | `/fab-clarify` (suggest + auto modes) |
| `<!-- clarified: {description} -->` | Resolved | `/fab-clarify` | Informational — not scanned |

**Placement**: Insert the marker inline in the artifact, immediately after the assumed or guessed content. The `{description}` MUST be a concise summary of what was assumed/guessed and why.

**Example**:
```markdown
The API SHALL return errors as JSON objects with `error`, `message`, and `code` fields.
<!-- assumed: JSON error format — config shows REST/JSON stack, consistent with existing patterns -->
```

### Assumptions Summary Block

Every planning skill invocation that makes Confident or Tentative assumptions SHALL end its output with an Assumptions summary and persist it as a trailing `## Assumptions` section in the generated artifact.

**Output format** (displayed to user):

```
## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | {decision summary} | {why this grade} |
| 2 | Tentative | {decision summary} | {why this grade} |

{N} assumptions made ({C} confident, {T} tentative). Run /fab-clarify to review.
```

**Artifact format** (persisted in the generated file): The same table is appended as the last section (`## Assumptions`) of the generated artifact. This ensures `/fab-clarify` can discover and scan assumptions from the artifact file.

**Rules**:
- Only include Confident and Tentative grades in the summary. Certain grades are omitted (not worth mentioning). Unresolved grades are asked as questions (not assumed).
- For `/fab-ff`, the output summary is **cumulative** across all generated stages. Each entry notes its source artifact (e.g., "in spec.md"). Per-artifact `## Assumptions` sections are persisted individually.
- If 0 assumptions were made, omit the Assumptions summary entirely (no empty table).

---

## Confidence Scoring

Confidence scoring provides a numeric measure of how well-resolved a change's decisions are, used as a gate for autonomous pipeline execution via `/fab-fff`.

### Schema (in `.status.yaml`)

```yaml
confidence:
  certain: 12      # count of Certain-graded SRAD decisions
  confident: 3     # count of Confident-graded decisions
  tentative: 2     # count of Tentative-graded decisions
  unresolved: 0    # count of Unresolved-graded decisions
  score: 2.1       # derived score (see formula below)
```

### Formula

```
if unresolved > 0:
  score = 0.0
else:
  score = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)
```

Range: 0.0 (any Unresolved, or 5+ Tentative) to 5.0 (all Certain). Penalties: Certain 0, Confident 0.3, Tentative 1.0, Unresolved → hard zero.

### Gate Threshold

`/fab-fff` requires `confidence.score >= 3.0`. This allows at most 2 Tentative decisions, or up to 6 Confident decisions with no Tentative.

### Invocation

Confidence is computed by `fab/.kit/scripts/_fab-score.sh`, invoked by `/fab-continue` (spec stage) and `/fab-clarify` (suggest mode). Autonomous skills (`/fab-ff`, `/fab-fff`) do not recompute — the gate check uses the score from the last manual step.

### Template

`fab/.kit/templates/status.yaml` includes the confidence block initialized to zero counts and score 5.0. Template defaults persist until `/fab-continue` generates the spec and invokes `_fab-score.sh`.
