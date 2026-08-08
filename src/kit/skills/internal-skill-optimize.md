---
name: internal-skill-optimize
description: "Condense a skill to its core — remove verbosity and redundancy without losing critical functionality — and apply structural health checks (table-of-contents on files over 100 lines, one-level reference-depth audit)."
---

# Internal Skill Optimize

Condense a skill (or all skills) to their core — remove verbosity, redundant examples, and re-explained concepts without losing any critical functionality — and apply two structural health checks (table-of-contents on long files, one-level reference depth).

This skill runs two distinct kinds of pass with **different scoping rules**:

- **Content optimization** (the bloat-signal trim) — operates on the requested skill. A consumer-skill pass treats every `_*.md` partial as read-only reference context and never trims one as a side effect; a dedicated pass invoked explicitly with a partial's name (e.g. `/internal-skill-optimize _preamble`) legitimately applies the same content signals to that partial. In batch mode, content optimization likewise skips partials (structural checks below still cover them).
- **Structural checks** (TOC + reference depth) — operate on **all** skill files **including `_*.md` partials**, because a long partial with no TOC (e.g. a 700+ line `_cli-fab.md`) or a partial that chains references more than one level deep is a real structural defect the content rule would wrongly exempt. Structural checks only add a Contents block or report a depth finding — they never trim a partial's prose.

---

## Contents

- Arguments
- Pre-flight
- Analysis (per skill)
- Optimization Rules
- Execution
- Constraints

## Arguments

- **`<skill-name>`** *(optional)* — name of a single skill to optimize (e.g., `fab-new`, `fab-continue`, `_preamble`). Resolves to `src/kit/skills/{skill-name}.md`. Naming a `_*.md` partial explicitly requests a dedicated partial-optimization pass.
- If omitted, process **all** `.md` files in `src/kit/skills/` except the `_*.md` partials. The rule, not a list: in batch mode, every `_*.md` file is read-only shared reference context.

---

## Pre-flight

1. Read the `_*.md` partials (deployed to `.claude/skills/`) — derive the set by globbing `src/kit/skills/_*.md`, never from a hardcoded list (it drifts as partials are added). In a consumer or batch pass they are read-only shared reference context; in a dedicated partial pass, the explicitly named partial is the target and the others remain reference context. Anything fully defined in a partial does NOT need re-explaining inside individual skills.
2. If a specific skill was requested, verify the file exists. If not, STOP with: `Skill not found: src/kit/skills/{skill-name}.md`

---

## Analysis (per skill)

For each skill file, read it fully and evaluate against these bloat signals.

**Content signals** (apply to individual skills and to an explicitly named partial; in consumer and batch passes partials are read-only context — content signals never apply to them, though the structural signals below do):

| Signal | What to look for |
|--------|-----------------|
| **Redundant re-explanation** | Concepts already defined in a partial being re-stated (SRAD rules in `_srad.md`, confidence formula in `_cli-fab.md` § fab score, context loading layers and preflight behavior in `_preamble.md`, generation procedures in `_generation.md`). Replace with a brief reference. |
| **Transition narration** | "No longer" / "used to" / "supersedes {id}" / "removed in {version}" / change-ID archaeology woven into instruction prose. Rewrite as present truth: keep the don't-re-break-this force (the prohibition and its reason), drop the history. This extends FKF §3.3's present-truth rule from memory to skills; see `$(fab kit-path)/reference/fkf.md` §3.3. |
| **Excessive output examples** | Multiple full output blocks showing minor variations. Consolidate to 1 compact example + brief notes on how it varies. |
| **Obvious instructions** | Telling an LLM things it already knows (what articles are, how to generate slugs, "continue to Step N" transitions). Remove. |
| **Redundant argument docs** | Same information appearing in both the Arguments section and a Behavior step. Keep one, reference the other. |
| **Over-specified error tables** | Error cases that are already handled by preflight scripts or shared conventions. Keep only skill-specific errors. |
| **Verbose step narration** | Steps that could be a single sentence but are expanded into paragraphs with sub-bullets. Compress. |
| **Duplicate examples** | Multiple examples illustrating the same point. Keep the most illustrative one. |
| **Sibling duplication** | In any content pass, compare the analyzed file with the already-loaded `_*.md` partials for the same rule, guard, or procedure. Ownership rule: a file may state a rule it owns or point at the file that owns it — never both. Report the owner/consumer pair and recommend reducing the consumer to a pointer, or report an extraction candidate; do not rewrite either file under this signal. |

**Structural signals** (apply to **all** files **including `_*.md` partials** — these add/report structure, never trim prose):

| Signal | What to look for | Fix |
|--------|-----------------|-----|
| **Missing table of contents** | A file **over 100 lines** with no `## Contents` (or `## Table of Contents`) block near the top. Anthropic's guidance: long files need a TOC so a partial read (`head -100`) still reveals the full scope. | Insert a `## Contents` bullet list of the file's `##`-level section headings. Placement, in document order: frontmatter → H1 → the `_preamble.md` blockquote (if present) → **`## Contents`** → first section. List the section titles **verbatim** (identical to the actual `##` headings) — no prose, so the TOC can't drift from the headings. |
| **Reference depth > 1 level** | A file whose link/`helpers:` reference points to another file that *itself* points onward to a third file holding content the reader needs (SKILL → A → B). Claude may `head`-preview nested files and miss content. | **Report only — do not auto-restructure.** Flag the chain (`{file} → {mid} → {leaf}`) so the maintainer can flatten it (link the leaf directly from the top file). Restructuring moves content and is out of scope for an automatic write. |

---

## Optimization Rules

1. **Never remove functionality** — every behavioral step, error case, and decision point must survive. The goal is fewer words for the same logic.
2. **Preserve frontmatter exactly** — `name`, `description` fields are untouched.
3. **Preserve the H1 heading and context reference** — `# /skill-name` and the `_preamble.md` blockquote stay. A new `## Contents` block (TOC structural check) goes immediately after them.
4. **Reference shared docs instead of re-explaining** — e.g., replace a 10-line SRAD re-explanation with "Apply the SRAD framework (see `_srad.md`)."
5. **Merge small sequential steps** — if Step N and Step N+1 are always done together and total <5 lines, combine them.
6. **One illustrative output example max** — show the canonical happy-path format and use inline notes like `(if --switch: include branch line)` for variations. This applies only to illustrative examples. Output literals that the skill's own logic or a sibling skill greps or string-matches are contracts, not examples — never consolidate or reword them (e.g. `git-pr`'s five `✓ commit` / `✓ push` / `✓ pr` / `✓ meta` / `✓ status` lines and `git-pr-review`'s three `Fixed —` / `Deferred —` / `Skipped —` reply prefixes).
7. **Keep error tables** — but remove rows already covered by preflight or `_preamble.md`.
8. **Preserve tone** — imperative, technical, precise. Don't soften.
9. **TOC keeps in sync** — when a content trim adds/removes/renames a `##` section in a file that has (or now needs) a `## Contents` block, update the TOC in the same pass so it never drifts from the headings.
10. **Depth findings are report-only** — the reference-depth check never moves content; it surfaces the chain for the maintainer to flatten as a separate change.

---

## Execution

### Single skill mode

1. Read the skill file
2. Run the content bloat-signal analysis on the named skill (including an explicitly named `_*.md` partial) AND both structural checks (TOC, reference depth — these run on partials too)
3. Produce a **before/after line count**, a **summary of content changes** (what was cut and why, 1 line per change), a **TOC action** (added / already present / not needed under 100 lines), and any **duplication findings** or **depth findings** (both report-only)
4. Present the summary to the user with `AskUserQuestion`: "Apply these optimizations to {skill-name}?"
5. On approval, write the optimized file (content trim + TOC insertion). Duplication and depth findings are reported, never auto-applied.

### Batch mode (no argument)

1. Read all skill files, sorted by line count descending (biggest bloat first)
2. **Content** optimization skips files under 80 lines ("Already lean — skipped") AND skips `_*.md` partials. **Structural** checks (TOC > 100 lines; depth) run on **every** file regardless — including partials and sub-80-line files
3. For each file, produce the before/after line count, content change summary, TOC action, consumer↔partial duplication findings, and depth findings
4. After per-file analysis, compare the analyzed non-partial skill files with one another for cross-skill/twin duplication (for example, `fab-ff` ↔ `fab-fff`) and report duplicated-rule clusters. Together with the consumer↔partial findings from the per-file passes, all duplication findings are report-only; extraction or pointer reduction is a separate content-moving change.
5. Present a single consolidated summary table to the user:

```
| Skill | Before | After | Reduction | TOC | Depth findings |
|-------|--------|-------|-----------|-----|----------------|
```

6. Below the table, list every report-only duplication finding and depth chain (`{file} → {mid} → {leaf}`) as follow-up candidates — these are NOT applied by this skill.
7. Ask: "Apply all optimizations, or select specific skills?"
8. On approval, write all approved files (content trims + TOC insertions; duplication findings and depth chains left for the maintainer)

---

## Constraints

- DO NOT change the logical behavior of any skill
- DO NOT remove error handling or edge case coverage
- DO NOT merge skills or move content between skills (beyond referencing `_preamble.md`)
- **Content optimization** DO NOT trim a `_*.md` partial as a side effect of optimizing a consumer skill or running batch mode — partials are read-only reference context in those passes. A dedicated partial-optimization pass invoked explicitly with the partial's name is legitimate and applies the same content signals. **Structural checks always include partials**: a TOC insertion may add a `## Contents` block to a partial, and the depth check may *report* on a partial.
- The reference-depth check is **report-only** — it never restructures or moves content (flattening a deep chain is a separate, content-moving change).
- If a skill is already under 80 lines, report it as "Already lean — skipped" for **content** optimization — but still run the **structural** checks on it. (The TOC check is a no-op below 100 lines — it adds nothing — while the depth check applies at any length; so a sub-80-line file is only ever flagged for a deep reference chain, never for a missing TOC.)
