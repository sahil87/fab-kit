---
name: docs-hydrate-specs
description: "Identify structural gaps between memory and specs, propose concise additions back to specs with interactive confirmation."
---

# /docs-hydrate-specs

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Contents

- Purpose
- Arguments
- Pre-flight
- Context Loading
- Behavior
- Error Handling
- Key Properties

---

## Purpose

Detect structural gaps between `docs/memory/` and `docs/specs/` — topics memory covers but specs don't — and propose concise additions back to specs. Top 3 gaps ranked by impact, with exact markdown previews and per-gap user confirmation.

This is the reverse of hydrate (specs → memory): hydrate-specs flows memory → specs.

---

## Arguments

- **`[domain]`** *(optional)* — scope to a single memory domain. If omitted, scans all domains.

---

## Pre-flight

Both index files must exist (see Error Handling); if one is missing, STOP with `{path} not found. Run /fab-setup first.` — i.e. `docs/memory/index.md not found. Run /fab-setup first.` or `docs/specs/index.md not found. Run /fab-setup first.`

---

## Context Loading

Loads `docs/memory/index.md`, `docs/specs/index.md`, all memory files (or scoped domain), and all spec files. Does NOT require `.fab-status.yaml`, config, or constitution.

---

## Behavior

### Step 1: Build Topic Inventory (Memory)

Read domain indexes and memory files. Extract `##`/`###` headings and brief summaries → list of `(file_path, topic_heading, summary)` tuples.

### Step 2: Build Coverage Inventory (Specs)

Read spec files. Extract headings and inline key term mentions.

### Step 3: Cross-Reference for Gaps

A topic is a **structural gap** if no spec heading covers it AND no spec mentions its key terms inline. Exclude topics that are purely implementation detail.

### Step 4: Rank and Cap

Rank by impact (High: core behavioral rules, design decisions; Medium: supporting concepts; Low: implementation details). Take top 3. Note overflow count.

### Step 5: Present Gaps with Previews

For each gap:

```
### Gap {N}: {topic name}

**Source**: `{memory_file}` → {heading}
**Target**: `{spec_file}` → after {section}

**Preview**:
{exact markdown to insert — matching target file's tone, style, heading levels}

Add this to `{spec_file}`? (yes / no / done)
```

**No-target branch**: when no existing spec file is a suitable home for the gap, the Target line proposes a **new** spec file instead — `**Target**: docs/specs/{kebab-topic}.md (new file)` — and the preview shows the full proposed file content, matching sibling specs' tone. The same per-gap confirmation gates it; specs stay human-curated (a fab-kit design principle).

### Step 6: Interactive Confirmation

The handler accepts exactly the tokens Step 5 offers (plus one alias):

- **yes** → insert at the specified location (preserve existing content); for a no-target gap, create the proposed new spec file and add its row to `docs/specs/index.md` (the one index edit this skill makes)
- **no** → skip, show next
- **done** (alias: **skip rest**) → proceed to summary

### Step 7: Summary

`Hydrate-specs complete: {N} of {M} gaps applied.` + overflow note if applicable. No gaps: `No structural gaps found between memory and specs.`

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `docs/memory/index.md` missing | Abort with init guidance |
| `docs/specs/index.md` missing | Abort with init guidance |
| No memory domains found | "No memory domains found. Run /docs-hydrate-memory first." |
| No spec files found | "No spec files found in docs/specs/index.md." |
| Domain argument unmatched | "Domain '{name}' not found. Available: {list}" |
| Spec file write fails | Report error, continue to next gap |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Requires active change? | No |
| Idempotent? | Yes |
| Modifies specs? | Yes — only with per-gap confirmation |
| Requires config/constitution? | No |
