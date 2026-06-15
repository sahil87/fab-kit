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
- **Backfill mode** (`backfill` keyword, or dispatched by `/docs-reorg-memory`): Re-scans an existing `docs/memory/` tree for topic files that lack `description:` frontmatter and adds it — **body-preserving** (only prepends/edits frontmatter). Used to migrate a pre-fab-kit, hand-curated tree to the fab-kit convention so `fab memory-index` stops rendering `—` for every row. Unlike generate mode (which *creates* files from source-code gaps), backfill *adds frontmatter to existing* files.

Mode is determined automatically by argument type (ingest/generate) or by the explicit `backfill` keyword / reorg dispatch. Safe to run repeatedly — content is merged without duplication or overwriting manually-added content; backfill skips files that already have `description:`.

### Index Ownership

Index files (`index.md` at the root, domain, and sub-domain tiers) are **generated artifacts** — `fab memory-index` is their single writer. The one hand-curated field is the `description:` frontmatter (on topic files and on domain/sub-domain indexes). When a new domain or sub-domain is created, its `index.md` **stub** — only the `description:` frontmatter one-liner, nothing else — is created **before** `fab memory-index` runs; the command fills in the generated body and round-trips the description. Never hand-edit generated index rows or "Last Updated" cells. Both modes below follow this model.

> **Refuse-before-regen guard (destructive-loss).** Before any `fab memory-index` regeneration step below, consult `fab memory-index --check`: on **exit 2** (destructive loss — a curated description would regenerate to `—`, a tombstone row would drop, or a custom grouping would flatten), **refuse to regenerate** and surface the pointer `→ run /docs-reorg-memory to remediate (it relocates removal-history rows to _shared/removed-domains.md and backfills description: frontmatter via /docs-hydrate-memory) before regenerating.` (`/docs-reorg-memory` is the orchestrator for all three tier-2 categories — it relocates tombstone rows itself and dispatches *this* skill's backfill mode for the descriptions; backfill alone does NOT relocate tombstones.) This is the primary pre-fab-kit-tree entry point, so the guard protects the *first* regen of a legacy tree. **No-op on born-compatible fab-kit trees** — they are always exit 0/1, never 2, so the guard never fires (do not mistake it for dead code). It only ever fires on a pre-fab-kit tree reached via ingest/generate before the tree has been backfilled. (Backfill mode itself never destroys content — it only adds frontmatter — so when *it* runs the regen, the guard has by then become a no-op.)

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
- **`backfill`** *(keyword)* — routes to **Backfill mode** (see below). Also entered when `/docs-reorg-memory` dispatches this skill as its compatibility sub-agent.

### Argument Classification & Mode Routing

| Argument type | Detection | Mode |
|---|---|---|
| `backfill` keyword | First argument is the literal `backfill`, OR the invocation is a `/docs-reorg-memory` dispatch naming the operation | **Backfill** (re-scan existing tree for files missing `description:`) |
| No arguments | Empty list | **Generate** (scan from project root) |
| URL | `notion.so`, `notion.site`, `linear.app`, or `http(s)://` | **Ingest** |
| Markdown file | Path ends `.md` | **Ingest** |
| Folder | Resolves to existing directory | **Generate** |

**Mode disambiguation** — backfill is checked first: it is reached only by the explicit `backfill` keyword or a reorg dispatch, so it never collides with bare ingest/generate routing. The two are otherwise distinct by intent: **generate** *creates* memory files from source-code gaps; **backfill** *adds `description:` frontmatter to existing* memory files (no new content). All non-backfill arguments must classify to the same mode. **Mixed-mode → reject**: `Cannot mix ingest sources (URLs, .md files) with generate targets (folders). Run separately.`

**Backfill takes no extra arguments** — backfill is an independent re-scan of `docs/memory/` with no caller manifest (see Backfill Mode Step 1), so any positional argument after the `backfill` keyword is meaningless. If `backfill` is the first argument **and** any further argument follows, **reject**: `backfill takes no arguments — it re-scans docs/memory/ itself. Run /docs-hydrate-memory backfill with no further arguments.` (The reorg-dispatch form never supplies extra args — it names only the operation.)

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

## Backfill Mode Behavior

Backfill migrates an **existing** hand-curated `docs/memory/` tree (typically pre-fab-kit) to the convention `fab memory-index` depends on: each topic file leads with a `description:` frontmatter line. Without it, the generator (which reads descriptions exclusively from frontmatter) renders `—` for every row, wiping curated descriptions on the first regen. Backfill is the one-time fix. It is invoked directly (`/docs-hydrate-memory backfill`) or dispatched by `/docs-reorg-memory` as the second step of its compatibility orchestration.

> **Scope**: Backfill is a **pure frontmatter operation** — it adds `description:` to existing files and creates missing `description:`-only index stubs. It does NOT detect or relocate tombstone rows, flatten custom groupings, or move files; those structural concerns belong to `/docs-reorg-memory`. The body of every file is preserved byte-for-byte.

### Step 1: Re-scan `docs/memory/` (no caller manifest)

Backfill **walks `docs/memory/` itself** to find every topic file (a non-`index.md` `.md` file) lacking a `description:` frontmatter field — it does **not** receive a file list from its caller. This holds for both forms: the direct-user invocation and the reorg dispatch (reorg's prompt names the operation — "backfill this tree" — not the files). A file with no frontmatter, or frontmatter without a `description:` key, counts as missing (the same `frontmatter.Field` semantics `fab memory-index` uses). The walk is the loose, idempotent seam between the two independently-invocable skills.

### Step 2: Synthesize and write `description:` frontmatter (body-preserving)

For each discovered topic file missing `description:`:

1. Read the file's **own content** — Overview, first section, or `# H1` — and synthesize a concise one-line summary.
2. **Prefer a curated index row** where one maps to this file. If an existing hand-curated index file (e.g., a pre-fab-kit `index.md` whose rows line up file-by-file with the topic files) has a row whose description text describes this file, use that curated text as the source — it is higher quality than re-synthesis.
3. Write the `description:` as the **leading frontmatter block** of the file (the same `---\ndescription: "..."\n---` shape ingest/generate use). **Preserve the body byte-for-byte** — backfill only prepends or edits the frontmatter, never the content below it.
4. **Skip files that already have a `description:`** — backfill never overwrites an existing one. This makes a second pass a no-op (idempotency, Constitution III).

### Step 3: Create missing index stubs (stub-before-index)

For any domain/sub-domain folder lacking an `index.md` (or whose `index.md` lacks `description:`), create the `description:`-only `index.md` **stub** the same way ingest/generate modes do — only the `description:` frontmatter one-liner, nothing else, created **before** any index regeneration (see Index Ownership above). This gives `fab memory-index` the domain description to read.

### Step 4: Caller-aware index regeneration

Backfill is **caller-aware** about `fab memory-index`:

- **Dispatched by `/docs-reorg-memory`** (the dispatch prompt carries the reorg-dispatched / defer-regen signal): do **NOT** run `fab memory-index`. reorg runs it exactly once at the end of its orchestration (after rebalance), so a regen here would be redundant work and would race reorg's single regen.
- **Invoked directly by a user** (no reorg signal): run `fab memory-index` as the final step, exactly like ingest and generate modes — root (domains-only) + every domain + every sub-domain index, regenerated from folder contents + frontmatter + git dates.

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

Backfill mode reports the re-scan and per-file frontmatter additions, e.g.:

```
Scanning docs/memory/ for files missing description: frontmatter...
Backfilled: docs/memory/{domain}/{topic}.md   (description: added; body unchanged)
Skipped:    docs/memory/{domain}/{other}.md   (already has description:)
Backfill complete — {N} files backfilled, {M} skipped, {S} index stubs created.
```

When dispatched by reorg, backfill appends `(index regen deferred to caller)`; when invoked directly, it runs `fab memory-index` and appends the regenerated-index lines like the other modes.

---

## Idempotency

Safe to re-run. New files created on first run, merged on subsequent. Existing content preserved. Indexes are regenerated by `fab memory-index` (byte-stable — a re-run with no content change produces no index diff). `[INFERRED]` markers and manual edits to memory files survive re-generation; index files are generated artifacts and are not hand-edited.

**Backfill mode** is idempotent on file presence of `description:`: files that already carry a `description:` field are skipped, so a second backfill pass over an already-converted tree is a no-op (no frontmatter rewrites, no body changes, byte-stable index). Backfill never touches a file's body — only its leading frontmatter — so re-running cannot corrupt or lose curated content.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `docs/memory/` or `docs/memory/index.md` missing | Abort with init guidance |
| Mixed-mode arguments | Reject with explanation |
| `backfill` keyword followed by extra arguments | Reject: "backfill takes no arguments — it re-scans docs/memory/ itself. Run /docs-hydrate-memory backfill with no further arguments." |
| Folder path doesn't exist | Abort: "Folder not found: {path}" |
| Source URL unreachable / content unreadable | Report error, continue with remaining |
| Domain/file already exists | Use/merge (don't recreate) |
| Backfill: file already has `description:` | Skip (idempotent) — never overwrite an existing description |
| Backfill: every topic file already has `description:` | Report `No files missing description: frontmatter — tree is already on the convention.` and stop (no regen when reorg-dispatched; a direct invocation may still run `fab memory-index`, which is a no-op) |

---

Next: {per state table — initialized}
