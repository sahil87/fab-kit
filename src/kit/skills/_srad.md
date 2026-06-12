---
name: _srad
description: "SRAD autonomy framework — decision scoring, confidence grades, artifact markers, and the Assumptions Summary block used by planning skills."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# SRAD Autonomy Framework

> This file defines the SRAD decision framework used by the planning skills
> (`fab-new`, `fab-draft`, `fab-continue`, `fab-ff`, `fab-fff`, `fab-clarify`), each of which
> declares `_srad` in its frontmatter `helpers:` list (see `_preamble.md` § Skill Helper Declaration).

When generating artifacts, planning skills encounter decision points not explicitly addressed by user input. The SRAD framework provides a principled method for deciding when to ask, when to assume, and when to surface assumptions.

---

## SRAD Scoring

For each decision point, evaluate four dimensions on a **continuous 0–100 scale** (100 = fully safe to assume, 0 = must ask):

| Dimension | High (75–100) | Medium (40–74) | Low (0–39) |
|-----------|--------------|----------------|------------|
| **S — Signal Strength** | Detailed description, multiple sentences, clear intent | Moderate detail, some gaps | One-liner, vague phrase, ambiguous scope |
| **R — Reversibility** | Easily changed later via `/fab-clarify` or stage reset | Moderate rework, a few files | Cascades through multiple artifacts, expensive to undo |
| **A — Agent Competence** | Config, constitution, codebase give clear answer | Partial signals, some inference | Business priorities, user preferences, political context |
| **D — Disambiguation Type** | One obvious default interpretation | 2–3 options, clear front-runner | Multiple valid interpretations with different tradeoffs |

**Aggregation**: Compute a composite score via weighted mean: `composite = 0.25*S + 0.30*R + 0.25*A + 0.20*D`. Map to grade using half-open thresholds (composites are continuous — 59.85 and 84.5 must grade deterministically): composite ≥ 85 → Certain, ≥ 60 → Confident, ≥ 30 → Tentative, else Unresolved. Critical Rule override: R < 25 AND A < 25 → always Unresolved.

Record per-dimension scores in the Assumptions table's required `Scores` column (e.g., `S:75 R:80 A:65 D:70`). The Scores column is mandatory for every row. `fab score` parses these and writes aggregate dimension statistics to `.status.yaml`.

## Confidence Grades

Each decision produces an assumption graded on a 4-level scale:

| Grade | Meaning | Artifact Marker | Output Visibility |
|-------|---------|----------------|-------------------|
| **Certain** | Determined by config/constitution/template rules | None | Noted in Assumptions summary |
| **Confident** | Strong signal, one obvious interpretation | None | Noted in Assumptions summary |
| **Tentative** | Reasonable guess, multiple valid options | `<!-- assumed: {description} -->` | Noted in Assumptions summary, `/fab-clarify` suggested |
| **Unresolved** | Cannot determine, incompatible interpretations | None — always asked or bailed | Asked as question AND noted in Assumptions summary |

## Critical Rule

**Decisions with R < 25 AND A < 25 (the override in the aggregation rule above — the Critical Rule's single numeric definition) are always Unresolved and MUST always be asked** — even in `/fab-new` and `/fab-continue`. These count toward the skill's question budget (max ~3). The existence of `/fab-clarify` as an escape valve does NOT justify silently assuming high-blast-radius decisions. `/fab-clarify` is for Tentative assumptions, not for Unresolved ones.

**Promptless-dispatch carve-out**: when a planning skill runs as a promptless subagent under `/fab-proceed`'s defer-and-surface contract (`fab-proceed.md` § fab-new Dispatch), there is no user to ask. The MUST-ask is satisfied by **deferring and surfacing**, never by silently assuming: each would-be-asked Unresolved decision is recorded as an Unresolved row with Rationale `Deferred — promptless dispatch` and surfaced to the user by the dispatcher. The intake gate is the structural backstop — `fab score` returns 0.0 whenever any Unresolved row exists, so a deferred decision always blocks the automated bracket until resolved via `/fab-clarify`. Everywhere a user is reachable, the MUST-ask applies unchanged.

## Skill-Specific Autonomy Levels

| Aspect | fab-new (adaptive) | fab-continue (deliberate) | fab-fff (full pipeline) | fab-ff (fast-forward) |
|--------|-------------------|---------------------------|-------------------------|--------------------------|
| **Posture** | SRAD-driven: 0 questions for clear inputs, conversational for vague; gap analysis before folder creation | SRAD at intake only (the one asking stage); apply decides-and-records | Gated on confidence; extends through ship + review-pr | Gated on confidence; stops at hydrate |
| **Interruption budget** | SRAD-driven (no fixed cap); conversational mode for vague inputs | 1-2 at intake; 0 at apply and later | 0 (autonomous rework, then stop) | 0 (autonomous rework, then stop) |
| **Output** | Assumptions summary + "Run /fab-clarify to review" | Key Decisions block + Assumptions summary | Cumulative Assumptions summary + apply/review/hydrate/ship/review-pr output | Tasks + apply/review/hydrate output |
| **Escape valve** | `/fab-clarify` | `/fab-clarify` | `/fab-clarify`, `/fab-continue` (after rework cap) | `/fab-clarify`, `/fab-continue` (after rework cap) |
| **Recomputes confidence?** | Yes (intake, via `fab score --stage intake`) | No (no scoring at apply — intake is authoritative) | No | No |

The remaining two declaring skills are covered by these columns: **fab-draft** follows the fab-new column exactly (it is a thin delta over fab-new Steps 0–9 — same SRAD-driven posture and budget; it only skips activation/branch). **fab-clarify** is the escape valve itself: suggest-mode questions are SRAD-prioritized (max 5 per invocation), resolved assumptions are re-graded in the artifact's table, and it always recomputes the intake score (`fab score --stage intake`).

## Worked Examples

### Example 1: Auth provider selection

> "Add auth." Which provider — OAuth2, SAML, API keys? → S/R/A/D: all Low (one word, no mechanism detail; auth architecture cascades into DB schema, middleware, API contracts; provider relationships are a user preference; multiple valid options with different tradeoffs). **Unresolved** — MUST be asked (Critical Rule applies: R < 25 AND A < 25).

### Example 2: Error response format

> "Handle errors" in a REST API → S: Medium, R/A/D: High (S:55 R:80 A:85 D:90 → composite 77). **Confident** — codebase signal is strong, easily reversed, one obvious default. Note in Assumptions summary, don't ask.

### Example 3: Test framework selection

> "Which test framework?" → S: Medium (terse but unambiguous in scope), R/A/D: High (S:50 R:95 A:100 D:100 → composite 86). **Certain** — config deterministically answers this (use existing runner). No marker; recorded in the Assumptions summary like every graded decision.

## Artifact Markers

Planning skills use HTML comment markers to flag assumptions for downstream scanning by `/fab-clarify`:

| Marker | Grade | Placed by | Scanned by |
|--------|-------|-----------|------------|
| `<!-- assumed: {description} -->` | Tentative | All planning skills (fab-new, fab-draft, fab-continue, fab-ff, fab-fff, fab-clarify) | `/fab-clarify` (suggest + auto modes) |
| `<!-- clarified: {description} -->` | Resolved | `/fab-clarify` | Informational — not scanned |

**Placement**: Insert the marker inline in the artifact, immediately after the assumed or guessed content. The `{description}` MUST be a concise summary of what was assumed/guessed and why.

**Example**:
```markdown
The API SHALL return errors as JSON objects with `error`, `message`, and `code` fields.
<!-- assumed: JSON error format — config shows REST/JSON stack, consistent with existing patterns -->
```

## Assumptions Summary Block

Every planning skill invocation SHALL include an Assumptions summary as the final content block of its output — immediately before the closing `Next:` line required by `_preamble.md` § Next Steps Convention — and persist it as a trailing `## Assumptions` section in the generated artifact.

**Output format** (displayed to user):

```
## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | {decision summary} | {why this grade} | S:nn R:nn A:nn D:nn |
| 2 | Confident | {decision summary} | {why this grade} | S:nn R:nn A:nn D:nn |
| 3 | Tentative | {decision summary} | {why this grade} | S:nn R:nn A:nn D:nn |
| 4 | Unresolved | {decision summary} | {status context} | S:nn R:nn A:nn D:nn |

{N} assumptions ({Ce} certain, {Co} confident, {T} tentative, {U} unresolved). Run /fab-clarify to review.
```

**Artifact format** (persisted in the generated file): The same table is appended as the last section (`## Assumptions`) of the generated artifact. This ensures `/fab-clarify` can discover and scan assumptions from the artifact file.

**Rules**:
- **Intake artifacts** include all four grades (Certain, Confident, Tentative, Unresolved). **plan.md `## Assumptions` excludes Unresolved** — three grades only, since apply decides-and-records (Unresolved is an intake-only construct). The Scores column (`S:nn R:nn A:nn D:nn`) is required for every row.
- Unresolved rows MUST include status context in the Rationale column: `Asked — {outcome}` or `Deferred — {reason}`.
- For `/fab-ff`, the output summary is **cumulative** across all generated stages. Each entry notes its source artifact (e.g., "in plan.md"). Per-artifact `## Assumptions` sections are persisted individually.
- **Omit-when-zero applies to the displayed output only**: if 0 assumptions were made, omit the Assumptions summary from the skill's output (no empty table). Generated artifacts (intake.md, plan.md) ALWAYS carry the `## Assumptions` section — the templates scaffold it and generators keep it; when empty, write the footer `0 assumptions.` with no table rows. This keeps `fab score` parsing uniform across artifacts.
