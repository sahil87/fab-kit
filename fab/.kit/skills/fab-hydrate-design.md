---
name: fab-hydrate-design
description: "Identify structural gaps between docs and specs, propose concise additions back to specs with interactive confirmation."
---

# /fab-hydrate-design

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Detect structural gaps between `fab/docs/` and `fab/design/` — topics that docs cover but specs don't mention at all — and propose concise additions back to specs. Presents the top 3 gaps ranked by impact, with exact markdown previews and per-gap user confirmation before writing anything.

This is the reverse of archive hydration (in `/fab-continue`): where archive flows specs → docs, hydrate-design flows docs → specs.

---

## Arguments

- **`[domain]`** *(optional)* — scope the comparison to a single doc domain (e.g., `fab-workflow`). If omitted, scans all domains.

---

## Pre-flight Check

Before doing anything else:

1. Check that `fab/docs/index.md` exists and is readable
2. Check that `fab/design/index.md` exists and is readable

**If either check fails, STOP immediately.** Output this message and do nothing else:

> `fab/docs/index.md not found. Run /fab-init first.`

or

> `fab/design/index.md not found. Run /fab-init first.`

---

## Context Loading

This skill loads:

1. `fab/docs/index.md` — to discover all doc domains
2. `fab/design/index.md` — to discover all spec files
3. All doc files across all domains (or scoped domain if argument provided) — read each file to build the topic inventory
4. All spec files listed in the specs index — read each file to build the coverage inventory

This skill does **not** require `fab/current`, `fab/config.yaml`, or `fab/constitution.md`.

---

## Behavior

### Step 1: Build Topic Inventory (Docs Side)

1. Read `fab/docs/index.md` to get the list of domains
2. If a `[domain]` argument was provided, filter to that domain only
3. For each domain, read the domain index (`fab/docs/{domain}/index.md`) to get doc files
4. For each doc file, extract:
   - The doc file path
   - All `## ` and `### ` level headings (these are "topics")
   - A brief summary of what each topic covers (first sentence or requirement text)

Result: a list of `(doc_path, topic_heading, summary)` tuples.

### Step 2: Build Coverage Inventory (Specs Side)

1. Read `fab/design/index.md` to get the list of spec files
2. For each spec file, extract:
   - All `## ` and `### ` level headings
   - All inline mentions of key terms (skill names, concept names, behavioral rules)

Result: a set of covered topics and terms.

### Step 3: Cross-Reference for Structural Gaps

For each doc topic from Step 1, check whether the specs coverage inventory from Step 2 mentions it — either as a heading or as an inline reference within a section.

A topic is a **structural gap** if:
- No spec file has a heading that covers the topic
- No spec file mentions the topic's key terms (skill name, concept name) within any section

A topic is **NOT a gap** if:
- Any spec file mentions it, even briefly or within a broader section
- The topic is purely implementation detail (file paths, script internals) rather than design intent

### Step 4: Rank and Cap

1. Rank gaps by impact:
   - **High impact**: Core behavioral rules, design decisions, skill behaviors that affect how humans understand the system
   - **Medium impact**: Supporting concepts, configuration patterns
   - **Low impact**: Implementation details, edge cases
2. Take the top 3 gaps
3. If more than 3 gaps exist, note the overflow count for the summary

### Step 5: Present Gaps with Markdown Previews

For each gap (up to 3), present:

```
### Gap {N}: {topic name}

**Source**: `{doc_file_path}` → {heading}
**Target**: `{spec_file_path}` → after {existing_section}

**Preview** (what would be added):

```markdown
{exact markdown to insert}
```

Add this to `{spec_file_path}`? (yes / no / done)
```

The markdown preview MUST:
- Match the existing spec file's tone and style — short declarative statements
- Be concise — no verbose explanations, no redundant detail
- Fit naturally into the target spec file's structure
- Use the same heading levels and formatting patterns as surrounding content

### Step 6: Interactive Confirmation

For each gap, wait for user input:

- **yes** — insert the markdown into the target spec file at the specified location
- **no** — skip this gap, present the next one
- **done** / **skip rest** — stop presenting gaps, proceed to summary

When inserting:
- Add the markdown at the appropriate location in the target spec file
- Preserve existing content — insert, don't replace
- Maintain consistent formatting (blank lines between sections, heading levels)

### Step 7: Summary

After all gaps have been presented (or user stopped early):

```
Hydrate-design complete: {N} of {M} gaps applied to specs.
{K} additional gaps not shown.
```

If no gaps were found:

```
No structural gaps found between docs and specs.
```

---

## Output

### Gaps Found

```
Scanning fab/docs/ against fab/design/...

Found 5 structural gaps (showing top 3):

### Gap 1: Preflight Script

**Source**: `fab/docs/fab-workflow/preflight.md` → ## Requirements
**Target**: `fab/design/architecture.md` → after ## Directory Structure

**Preview** (what would be added):

### Preflight Script (`fab-preflight.sh`)

A shared validation script that verifies project initialization, active change existence, and `.status.yaml` integrity. Returns structured YAML output for skill consumption. Used by all skills that operate on an active change.

Add this to `fab/design/architecture.md`? (yes / no / done)
```

### No Gaps

```
Scanning fab/docs/ against fab/design/...

No structural gaps found between docs and specs.
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/docs/index.md` missing | Abort: "fab/docs/index.md not found. Run /fab-init first." |
| `fab/design/index.md` missing | Abort: "fab/design/index.md not found. Run /fab-init first." |
| No doc domains found | Output: "No doc domains found. Run /fab-hydrate first." |
| No spec files found | Output: "No spec files found in fab/design/index.md." |
| Domain argument doesn't match any domain | Output: "Domain '{name}' not found. Available: {list}" |
| Spec file write fails | Report error, continue to next gap |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — not part of the change lifecycle |
| Requires active change? | **No** — operates on project-level docs and specs |
| Idempotent? | **Yes** — safe to run multiple times; re-detects gaps each time |
| Modifies `fab/current`? | **No** |
| Modifies `.status.yaml`? | **No** |
| Modifies source code? | **No** |
| Modifies specs? | **Yes** — only with explicit per-gap user confirmation |
| Creates git branch? | **No** |
| Requires config/constitution? | **No** |
