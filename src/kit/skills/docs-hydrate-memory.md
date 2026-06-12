---
name: docs-hydrate-memory
description: "Hydrate memory from external sources or generate from codebase analysis. Safe to re-run."
---

# /docs-hydrate-memory [sources...|folders...]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Purpose

Hydrate `docs/memory/` from external sources or from codebase analysis.

- **Ingest mode** (URLs, `.md` files): Fetches/reads sources, identifies domains and topics, creates or merges memory files, maintains indexes.
- **Generate mode** (folders, no arguments): Scans codebase for undocumented areas, presents interactive gap report, generates memory files.

Mode is determined automatically by argument type. Safe to run repeatedly — content is merged without duplication or overwriting manually-added content.

### Index Ownership

Index files (`index.md` at the root, domain, and sub-domain tiers) are **generated artifacts** — `fab memory-index` is their single writer. The one hand-curated field is the `description:` frontmatter (on topic files and on domain/sub-domain indexes). When a new domain or sub-domain is created, its `index.md` **stub** — only the `description:` frontmatter one-liner, nothing else — is created **before** `fab memory-index` runs; the command fills in the generated body and round-trips the description. Never hand-edit generated index rows or "Last Updated" cells. Both modes below follow this model.

---

## Pre-flight Check

1. `docs/memory/` directory must exist
2. `docs/memory/index.md` must exist and be readable

**If either fails, STOP**: `docs/memory/ not found. Run /fab-setup first to create the memory directory.` Do NOT create these.

---

## Context Loading

Skips the always-load layer entirely (this section is the skill-file override the `_preamble.md` §1 contract keys on): the skill ingests or generates memory content — it does not need to pre-load the memory landscape, and it requires no config, constitution, or active change. Up-front, only the Pre-flight files above are read — the skill's own working inputs (ingest sources in Step 1, scanned codebase files in generate mode) are still read during execution; what is skipped is the always-load layer, nothing more.

---

## Arguments

- **`[sources...|folders...]`** *(optional)* — zero or more URLs, local `.md` paths, or folder paths.

### Argument Classification & Mode Routing

| Argument type | Detection | Mode |
|---|---|---|
| No arguments | Empty list | **Generate** (scan from project root) |
| URL | `notion.so`, `notion.site`, `linear.app`, or `http(s)://` | **Ingest** |
| Markdown file | Path ends `.md` | **Ingest** |
| Folder | Resolves to existing directory | **Generate** |

All arguments must classify to the same mode. **Mixed-mode → reject**: `Cannot mix ingest sources (URLs, .md files) with generate targets (folders). Run separately.`

Folder paths must exist — abort with `Folder not found: {path}` if not.

---

## Ingest Mode Behavior

### Step 1: Fetch/Read Source Content

- **Notion URL**: Fetch via MCP tool/API. Extract title and body.
- **Linear URL**: Fetch via MCP tool/API. Extract title, description, details.
- **Local path**: Read directly. If directory, recursively read all `.md` files.

Report: `Fetched: {title or filename} ({source type})`

### Step 2: Analyze and Map to Domains

For each source: identify **domains** (logical topic areas) and **topics** within each. Map to target files: `docs/memory/{domain}/{topic}.md`.

### Step 3: Create or Merge Memory Files

For each topic:
1. Create `docs/memory/{domain}/` if needed
2. Create `docs/memory/{domain}/index.md` if needed — a stub carrying only the `description:` frontmatter one-liner for the domain, created before Step 4 runs (`fab memory-index` reads it into the root index row — see Index Ownership). When placing a topic into a sub-domain, likewise create the `docs/memory/{domain}/{sub-domain}/index.md` stub if needed
3. If target file doesn't exist → create with a leading `description:` frontmatter line, then Overview, Requirements, Design Decisions, Changelog sections
4. If target file exists → **merge** new content, preserve existing/manually-added content; keep its `description:` frontmatter accurate

**Author the `description:` frontmatter** on every file you create or whose summary changes — it is the source for the generated index row (Step 4). Do NOT hand-write index rows.

**Shape bounds (SHOULD guidance)** when placing topics into domains:
- Aim for **~5–12 topic files per folder**. Past ~12, `fab memory-index` warns — consider a sub-domain.
- **Max depth 3**: `docs/memory/{domain}/{sub-domain}/{topic}.md`.
- Introduce a sub-domain **only reactively**, when a cohesive cluster of **≥8 files** exists. Never pre-build hierarchy.
- Reserved domains `_shared/` (cross-cutting) and `_unsorted/` (staging) are exempt from the width bound.

### Step 4: Regenerate Indexes (`fab memory-index`)

Run `fab memory-index` once. It deterministically regenerates the root (domains-only), every domain index, and every sub-domain index from folder contents + `description:` frontmatter + git dates — byte-stable and idempotent. Never hand-edit index rows or "Last Updated" cells; the command is the single writer. Any non-fatal shape warnings it prints to stderr are advisory (over-wide / over-deep folders).

---

## Generate Mode Behavior

### Step 1: Codebase Scanning

Scan target scope (project root if no args, specified folders otherwise). Exclude `.git/`, `node_modules/`, `vendor/`, `__pycache__/`, `dist/`, `build/`.

Detect gaps in five categories:

1. **Modules**: Top-level source dirs without matching `docs/memory/` domains
2. **APIs**: Route definitions, endpoint handlers, CLI commands, exported interfaces not in memory
3. **Patterns**: Recurring structural patterns (3+ occurrences) without memory coverage
4. **Configuration**: Config files and env var references not documented
5. **Conventions**: File naming patterns, directory conventions (lowest priority)

Cross-reference against existing memory — exclude already-covered areas.

### Step 2: Gap Report & Interactive Scoping

**Zero gaps**: Output `No memory gaps found. docs/memory/ is up to date.` and stop.

**Gap report format** (grouped by category with priorities):

```
## Memory Gap Report

### Modules
1. [High] auth module — src/auth/
2. [Medium] utils — src/utils/

### APIs
3. [High] REST API endpoints — src/api/routes/
```

**4+ gaps**: Use AskUserQuestion with options: "All", "All High priority", "Select by number", "Select by category".

**1-3 gaps**: Confirm: `Found {N} undocumented area(s). Document all?`

### Step 3: Memory File Generation

For each selected gap: read **all source files** in scope, synthesize into **one memory file per gap** using this format:

```markdown
---
description: "One-line summary of this topic (source for the generated index row)."
---
# {Topic}

## Overview
{What it does, inferred from code.}

## Requirements
{Key behaviors as RFC 2119 requirements. Derived from code, not invented.}

## Design Decisions
{Architectural choices with rationale where inferable.}

## Changelog
| Date | Change |
|------|--------|
| {DATE} | Generated from code analysis |
```

Mark ambiguous inferences with `[INFERRED]` inline near the relevant requirement.

**Placement** follows the same rules as ingest-mode Step 3: target path is `docs/memory/{domain}/{topic}.md` (or `docs/memory/{domain}/{sub-domain}/{topic}.md`); create the domain folder and its `description:`-only index stub if needed — sub-domain stub likewise — before Step 4 runs (see Index Ownership); and the same shape bounds apply (~5–12 topic files per folder, max depth 3, a sub-domain only for a cohesive ≥8-file cluster, `_shared/`/`_unsorted/` width-exempt).

### Step 4: Regenerate Indexes

Same as ingest mode Step 4 — run `fab memory-index` to regenerate the root (domains-only), domain, and sub-domain indexes from folder contents + frontmatter + git dates. Do not hand-edit index rows.

---

## Output

Canonical format (ingest mode):

```
Hydrating memory from {N} source(s)...
Fetched: {title} ({source type})
Created: docs/memory/{domain}/{topic}.md
Updated: docs/memory/{domain}/index.md   (via fab memory-index)
Updated: docs/memory/index.md            (via fab memory-index)
Hydration complete — {N} files created, {M} updated.
```

Generate mode replaces "Hydrating" with "Scanning codebase for memory gaps..." and includes the gap report before generation output. Re-hydration shows "merged new content" for updated files. Zero gaps stops after the scan summary.

---

## Idempotency

Safe to re-run. New files created on first run, merged on subsequent. Existing content preserved. Indexes are regenerated by `fab memory-index` (byte-stable — a re-run with no content change produces no index diff). `[INFERRED]` markers and manual edits to memory files survive re-generation; index files are generated artifacts and are not hand-edited.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `docs/memory/` or `docs/memory/index.md` missing | Abort with init guidance |
| Mixed-mode arguments | Reject with explanation |
| Folder path doesn't exist | Abort: "Folder not found: {path}" |
| Source URL unreachable / content unreadable | Report error, continue with remaining |
| Domain/file already exists | Use/merge (don't recreate) |

---

Next: {per state table — initialized}
