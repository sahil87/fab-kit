# Shared Context Preamble

> This file defines shared conventions for all Fab skills. Each skill file should begin with:
> `Read and follow the instructions in fab/.kit/skills/_context.md before proceeding.`

---

## Context Loading

Before generating or validating any artifact, load the relevant context layers below. This ensures output is grounded in the actual project state, not assumptions.

### 1. Always Load (every skill except `/fab-init`, `/fab-status`, `/fab-hydrate`; `/fab-switch` loads only `config.yaml`)

Read these files first — they define the project's identity, constraints, and documentation landscape:

- **`fab/config.yaml`** — project configuration, tech stack, naming conventions, stage configuration
- **`fab/constitution.md`** — project principles and constraints (MUST/SHOULD/MUST NOT rules)
- **`fab/docs/index.md`** — documentation landscape (which domains and docs exist)
- **`fab/specs/index.md`** — specifications landscape (pre-implementation design intent, human-curated)

> **Note**: If the skill runs `fab-preflight.sh` (Section 2 above), the init check (config.yaml and constitution.md existence) is already covered by the script. Skills using preflight don't need separate existence checks for these files — they only need to read them for content.

### 2. Change Context (when operating on an active change)

Resolve the active change and load its state by running the preflight script:

1. **Run preflight**: Execute `fab/.kit/scripts/fab-preflight.sh` via Bash
2. **Check exit code**: If the script exits non-zero, STOP and surface the stderr message to the user (it contains the specific error and suggested fix)
3. **Parse stdout YAML**: On success, parse the YAML output for `name`, `change_dir`, `stage`, `progress`, `checklist`, and `confidence` fields — use these for all subsequent change context instead of re-reading `.status.yaml`
4. Load all completed artifacts in the change folder (e.g., `proposal.md`, `spec.md`, `plan.md`, `tasks.md`) — read each file that exists so you have full context of what has been decided so far

> **What the script validates internally** (for reference — agents do not need to duplicate these checks):
> 1. `fab/config.yaml` and `fab/constitution.md` exist (project initialized)
> 2. `fab/current` exists and is non-empty (active change set)
> 3. Change directory `fab/changes/{name}/` exists
> 4. `.status.yaml` exists within the change directory

### 3. Centralized Doc Lookup (when operating on an active change)

Selectively load relevant domain docs based on the change's scope:

1. Read the proposal's **Affected Docs** section (or spec's **Affected docs** metadata) to identify which domains are relevant
2. For each referenced domain, read `fab/docs/{domain}/index.md` to understand the domain's docs
3. Read the specific centralized doc(s) referenced by the Affected Docs entries (the New, Modified, and Removed entries) — read `fab/docs/{domain}/{name}.md` for each listed doc that exists
4. If a referenced doc or domain does not exist yet (e.g., listed under New Docs), note this and proceed without error — it will be created by `/fab-archive`
5. Use this context to ground all artifact generation (specs, plans, tasks, reviews) in the real current state, not assumptions

### 4. Source Code Loading (during implementation and review)

Load only the source files relevant to the current work:

1. Read the relevant source files referenced in the plan's **File Changes** section (New, Modified, Deleted) or in the task descriptions
2. Scope to files actually touched by the change — do not load the entire codebase
3. This applies primarily to `/fab-apply` and `/fab-review`

---

## Next Steps Convention

Every skill MUST end its output with a `Next:` line suggesting the available follow-up commands. This keeps the user oriented in the workflow without needing to memorize the stage graph.

**Format**: `Next: /fab-command` or `Next: /fab-commandA or /fab-commandB (description)`

### Lookup Table

| After skill | Stage reached | Next line |
|-------------|---------------|-----------|
| `/fab-init` | initialized | `Next: /fab-new <description>, /fab-discuss <idea>, or /fab-hydrate <sources>` |
| `/fab-hydrate` | docs hydrated | `Next: /fab-new <description>, /fab-discuss <idea>, or /fab-hydrate <more-sources>` |
| `/fab-new` | proposal done | `Next: /fab-continue or /fab-ff (fast-forward all planning)` |
| `/fab-discuss` (new, activated) | proposal done | `Next: /fab-continue or /fab-ff (fast-forward all planning)` |
| `/fab-discuss` (new, not activated) | proposal done | `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-ff` |
| `/fab-discuss` (refined) | proposal updated | `Next: /fab-continue or /fab-ff (fast-forward all planning)` |
| `/fab-continue` → specs | specs done | `Next: /fab-continue (plan) or /fab-ff (fast-forward) or /fab-clarify (refine spec)` |
| `/fab-continue` → plan | plan done | `Next: /fab-continue (tasks) or /fab-clarify (refine plan)` |
| `/fab-continue` → tasks | tasks done | `Next: /fab-apply` |
| `/fab-ff` | tasks done | `Next: /fab-apply` |
| `/fab-clarify` | same stage | `Next: /fab-clarify (refine further) or /fab-continue or /fab-ff` |
| `/fab-apply` | apply done | `Next: /fab-review` |
| `/fab-review` (pass) | review done | `Next: /fab-archive` |
| `/fab-review` (fail) | review failed | *(contextual — see /fab-review for fix options)* |
| `/fab-fff` | archived | `Next: /fab-new <description> (start next change)` |
| `/fab-fff` (bail) | varies | *(contextual — see /fab-fff for bail messages)* |
| `/fab-archive` | archived | `Next: /fab-new <description> (start next change)` |

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

### Extending the Protocol

If future skills need additional mode signals, define new bracketed prefixes (e.g., `[BATCH-MODE]`) in this section. The pattern is: one prefix per mode, first-line placement, absence means default.

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

| Aspect | fab-discuss (explore) | fab-new (capture) | fab-continue (deliberate) | fab-ff (speed) | fab-fff (full pipeline) |
|--------|----------------------|-------------------|---------------------------|----------------|-------------------------|
| **Posture** | Free-form conversation, gap analysis, no question cap | Assume confident+tentative, ask top ~3 unresolved | Surface tentative, ask top ~3 unresolved | Batch all unresolved upfront, then go | Same as fab-ff; gated on confidence >= 3.0 |
| **Interruption budget** | Unlimited — conversational by design | Max 3 for unresolved questions | 1-2 per stage | 0-1 batch at start | Same as fab-ff (frontloaded) |
| **Output** | Proposal + confidence score + "/fab-switch to make active" | Assumptions summary + "Run /fab-clarify to review" | Key Decisions block + Assumptions summary + [NEEDS CLARIFICATION] count | Cumulative Assumptions summary | Same as fab-ff + apply/review/archive output |
| **Escape valve** | User ends early at any time | `/fab-clarify` | `/fab-clarify` | `/fab-clarify` | `/fab-clarify` (bails on blockers or review failure) |
| **Recomputes confidence?** | Yes | Yes | Yes | No | No |

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

> **Decision point**: Spec says "handle errors." What format — JSON body, plain text, RFC 7807?
>
> | Dimension | Score | Reasoning |
> |-----------|-------|-----------|
> | S — Signal | Medium | "Handle errors" is vague, but the context is a REST API |
> | R — Reversibility | High | Error format is easily changed later without cascading |
> | A — Agent Competence | High | Config shows REST/JSON stack; existing codebase uses JSON errors |
> | D — Disambiguation | High | JSON error body is the obvious default for a REST API |
>
> **Grade: Confident** — strong codebase signal, easily reversed, one obvious choice. Note in Assumptions summary but do not ask.

#### Example 3: Test framework selection

> **Decision point**: Adding a new module. Which test framework — Jest, Vitest, project's existing runner?
>
> | Dimension | Score | Reasoning |
> |-----------|-------|-----------|
> | S — Signal | Low | User didn't mention testing approach |
> | R — Reversibility | High | Test files are self-contained; switching frameworks later is straightforward |
> | A — Agent Competence | High | `config.yaml` and `package.json` specify the existing test runner |
> | D — Disambiguation | High | Use whatever the project already uses |
>
> **Grade: Certain** — config deterministically answers this. No marker, no mention in Assumptions summary.

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
  score: 2.7       # derived score (see formula below)
```

### Formula

```
if unresolved > 0:
  score = 0.0
else:
  score = max(0.0, 5.0 - 0.1 * confident - 1.0 * tentative)
```

- **Range**: 0.0 to 5.0
- **5.0**: All decisions are Certain — maximum confidence
- **0.0**: Any Unresolved decision, OR 5+ Tentative decisions
- Certain contributes 0 penalty (deterministic, no ambiguity)
- Confident contributes 0.1 penalty (minor — strong signal, one obvious interpretation)
- Tentative contributes 1.0 penalty (meaningful — reasonable guess but multiple valid options)
- Unresolved is a hard zero (cannot run autonomously with unresolved decisions)

### Gate Threshold

`/fab-fff` requires `confidence.score >= 3.0`. This allows at most 2 Tentative decisions (with some Confident erosion).

### Lifecycle

| Event | Skill | Action |
|-------|-------|--------|
| Initial computation | `/fab-new` | Count SRAD grades across proposal, compute score, write to `.status.yaml` |
| Recomputation | `/fab-continue` | Re-count across all artifacts after generating each one, update `.status.yaml` |
| Recomputation | `/fab-clarify` | Re-count after each suggest-mode session, update `.status.yaml` |
| No recomputation | `/fab-ff`, `/fab-fff` | Autonomous skills do not update the score — gate check uses score from last manual step |
| Consumption | `/fab-fff` | Reads score as pre-flight gate check |

### Template

`fab/.kit/templates/status.yaml` includes the confidence block initialized to zero counts and score 5.0. `/fab-new` uses this template when creating new changes.
