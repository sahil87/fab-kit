---
name: fab-hydrate
description: "Hydrate memory from external sources or generate from codebase analysis. Safe to re-run."
---

# /fab-hydrate [sources...|folders...]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Hydrate `fab/memory/` from external sources or from codebase analysis.

- **Ingest mode** (URLs, `.md` files): Fetches or reads the provided sources, analyzes content to identify domains and topics, creates or merges memory files, and maintains indexes.
- **Generate mode** (folders, no arguments): Scans the codebase for undocumented areas, presents an interactive gap report, lets the user select which gaps to document, and generates structured memory files into `fab/memory/`.

Mode is determined automatically by argument type — no flags needed. Safe to run repeatedly — content is merged into existing memory files without duplication or overwriting manually-added content.

---

## Pre-flight Check

Before doing anything else:

1. Check that `fab/memory/` directory exists
2. Check that `fab/memory/index.md` exists and is readable

**If either check fails, STOP immediately.** Output this message and do nothing else:

> `fab/memory/ not found. Run /fab-init first to create the memory directory.`

Do NOT create `fab/memory/` or `fab/memory/index.md`. The structural bootstrap (`/fab-init`) must have run first.

---

## Arguments

- **`[sources...|folders...]`** *(optional)* — zero or more URLs, local markdown paths, or folder paths. The argument type determines the operating mode.

### Argument Classification

Classify each argument in order:

| Argument type | Detection rule | Mode |
|---|---|---|
| No arguments | Argument list is empty | **Generate** (scan from project root) |
| URL | Matches `notion.so`, `notion.site`, `linear.app`, or `http(s)://` | **Ingest** |
| Markdown file | Path ends with `.md` | **Ingest** |
| Folder | Path resolves to an existing directory | **Generate** |

### Mode Routing

1. If **no arguments** are provided → enter **generate mode**, scanning from the project root.
2. If **all arguments** classify as ingest sources (URLs or `.md` files) → enter **ingest mode**.
3. If **all arguments** classify as folders → enter **generate mode**, scoping the scan to those folders.
4. If arguments are a **mix** of ingest and generate types → **reject with error**:

> `Cannot mix ingest sources (URLs, .md files) with generate targets (folders). Run separately.`

No processing occurs on mixed-mode invocations.

### Folder Validation

For folder arguments, verify the path exists as a directory. If a folder path does not exist, report an error and abort:

> `Folder not found: {path}`

---

## Ingest Mode Behavior

When arguments route to **ingest mode**, execute the following steps for each source provided:

### Step 1: Fetch/Read Source Content

- **Notion URL** (contains `notion.so` or `notion.site`): Fetch page or database content via Notion MCP tool or API. Extract the page title and body content.
- **Linear URL** (contains `linear.app`): Fetch issue or project content via Linear MCP tool or API. Extract title, description, and relevant details.
- **Local file path**: Read the file content directly. If it's a directory, recursively read all `.md` files within it.

Report each source as it's fetched: `Fetched: {title or filename} ({source type})`

### Step 2: Analyze and Map to Domains

For each fetched source:

1. Analyze the content to identify **domains** (logical topic areas, e.g., `auth`, `payments`, `api`, `infrastructure`)
2. Identify individual **topics** within each domain (e.g., `authentication`, `authorization` within `auth`)
3. Map each topic to a target file: `fab/memory/{domain}/{topic}.md`

### Step 3: Create or Merge Memory Files

For each identified topic:

1. If `fab/memory/{domain}/` directory does not exist, create it
2. If `fab/memory/{domain}/index.md` does not exist, create a domain index following the [Domain Index format](../../specs/templates.md#domain-index-fabmemorydomainindexmd):
   ```markdown
   # {Domain} Memory

   | File | Description | Last Updated |
   |------|-------------|-------------|
   ```
3. If the target memory file does not exist, create it with the analyzed content, structured as a [Memory File](../../specs/templates.md#individual-file-fabmemorydomainnamemd) (with Overview, Requirements sections, Design Decisions, Changelog)
4. If the target memory file already exists, **merge** the new content into the existing file — add new requirements, update existing ones, do not remove existing content. Preserve any manually-added content.

### Step 4: Update Domain Indexes

For each domain that was created or had files added:

1. Read `fab/memory/{domain}/index.md`
2. Add rows for any new files that were created in this domain
3. Update "Last Updated" column for any files that were modified
4. Use relative links: `| [{name}]({name}.md) | {description} | {DATE} |`

### Step 5: Update Top-Level Index

After all sources are processed:

1. Read the current `fab/memory/index.md`
2. Add rows for any new domains that were created: `| [{domain}]({domain}/index.md) | {description} | {file-list} |`
3. Update the file-list column for existing domains that had new files added (comma-separated list of all files in the domain)
4. Do not remove existing entries
5. Use relative links (not absolute paths) — see [Top-Level Index format](../../specs/templates.md#top-level-index-fabmemoryindexmd)

Report what was created and updated:
```
Created: fab/memory/{domain}/{topic}.md
Updated: fab/memory/{domain}/index.md
Updated: fab/memory/index.md
```

---

## Generate Mode Behavior

When arguments route to **generate mode** (no arguments, or folder paths), execute the following steps.

### Step 1: Codebase Scanning

Scan the target scope to identify undocumented areas. The scan target is:
- **No arguments**: the project root directory
- **Folder arguments**: only the specified folder(s) and their subdirectories

Use Glob, Grep, and Read tools to analyze the codebase. Exclude these directories from scanning: `.git/`, `node_modules/`, `vendor/`, `__pycache__/`, `dist/`, `build/`.

#### Detection by Category

Scan for gaps in five categories, using the following heuristics:

1. **Modules**: Enumerate top-level source directories. For each, check if a matching domain exists in `fab/memory/index.md`. Unmatched directories → Module gap.

2. **APIs**: Grep for route definitions, endpoint handlers, CLI command registrations, and exported public interfaces. Look for patterns like `app.get(`, `@route`, `export function`, `def command`, etc. Cross-reference against existing `fab/memory/` entries. Undocumented endpoints/exports → API gap.

3. **Patterns**: Identify recurring structural patterns across the codebase — middleware chains, plugin directories, event handler registrations, factory functions, decorator usage. If a pattern appears 3+ times and has no corresponding file in `fab/memory/`, flag it as a Pattern gap.

4. **Configuration**: Glob for config files (`.env*`, `*.config.*`, `config/`, `settings.*`) and grep for environment variable references (`process.env`, `os.environ`, `ENV[]`). Undocumented config → Configuration gap.

5. **Conventions**: Analyze file naming patterns, directory structure conventions, common prefixes/suffixes. These are lowest priority and only flagged when the pattern is clear and consistent.

#### Cross-Reference Against Existing Memory

For each potential gap found, check `fab/memory/` domains and their entries:
- If the area is already covered by an existing memory file → **exclude** it from the gap report (or deprioritize it)
- If the area is partially covered → include it with a note about what's missing

### Step 2: Gap Report & Interactive Scoping

#### Zero Gaps

If the scan finds no undocumented areas, output:

> `No memory gaps found. fab/memory/ is up to date.`

Then stop — do not present the selection UI or proceed to memory file generation.

#### Gap Report Format

Present a numbered gap report, grouped by category with priorities. Format:

```
## Memory Gap Report

### Modules
1. [High] auth module — src/auth/
2. [Medium] utils — src/utils/

### APIs
3. [High] REST API endpoints — src/api/routes/

### Patterns
4. [Medium] Middleware chain — src/middleware/*.ts (5 occurrences)

### Configuration
5. [Low] Environment variables — .env, .env.example

### Conventions
6. [Low] fab- prefix naming — fab/.kit/scripts/fab-*.sh
```

Each gap includes:
- **Category**: Module, API, Pattern, Configuration, or Convention
- **Name**: Human-readable identifier
- **Location**: File paths or directory paths involved
- **Priority**: High (core functionality, public API), Medium (internal modules, patterns), Low (utilities, config)

#### Interactive Selection (4+ gaps)

After presenting the gap report, use AskUserQuestion with these 4 strategy options:

1. **"All"** — Document everything found
2. **"All High priority"** — Document only High priority gaps
3. **"Select by number"** — User types gap numbers in the Other text input (e.g., "1, 3, 5")
4. **"Select by category"** — User types category names in the Other text input (e.g., "Modules, APIs")

Parse the user's selection and proceed to memory file generation with only the selected gaps.

#### Small Number of Gaps (1-3)

If 1-3 gaps are found, skip the AskUserQuestion UI. Instead, confirm:

> `Found {N} undocumented area(s). Document all?`

On confirmation, proceed to memory file generation for all of them.

### Step 3: Memory File Generation

For each selected gap, generate a memory file.

#### Reading Source Code

Before generating a memory file, read **all source files** within the gap's scope (directory, file set, or pattern matches). Synthesize the content into **one memory file per gap** — not one file per source file. For example, a module gap covering `src/auth/` produces a single memory file that covers the entire auth module, not separate files for each source file in the directory.

#### Memory File Format

Write each generated memory file to `fab/memory/{domain}/{topic}.md` using this format:

```markdown
# {Topic}

## Overview

{What the module/API/pattern does, inferred from code analysis.}

## Requirements

{Key behaviors documented as requirements using RFC 2119 keywords (MUST, SHALL, SHOULD, MAY).
Derive these from actual code behavior — do not invent requirements.}

## Design Decisions

{Architectural choices evident from the code, with rationale where inferable.}

## Changelog

| Date | Change |
|------|--------|
| {DATE} | Generated from code analysis |
```

#### `[INFERRED]` Markers

When behavior is ambiguous or uncertain from code alone, mark it with an `[INFERRED]` tag inline:

> `The auth module [INFERRED] uses JWT tokens for session management — verify this against actual token handling.`

Place markers close to the relevant requirement, not in a separate section. Include an explanation of what was inferred and suggest verification.

### Step 4: Index Maintenance

After generating memory files, maintain indexes using the **same logic as ingest mode Steps 4-5**:

1. Create or update `fab/memory/{domain}/index.md` for each domain touched
2. Update `fab/memory/index.md` with new domains and file lists
3. All links SHALL be relative
4. Existing entries SHALL NOT be removed

This step is shared between ingest and generate modes.

---

## Output

### Successful Hydration

```
Hydrating memory from {N} source(s)...
Fetched: {title} ({source type})
Fetched: {N} files from {directory path}
Created: fab/memory/{domain}/{topic}.md
Created: fab/memory/{domain}/{topic}.md
Updated: fab/memory/{domain}/index.md
Updated: fab/memory/index.md
Hydration complete — {N} files created, {M} updated.
```

### Multiple Sources

```
Hydrating memory from 3 source(s)...
Fetched: API Spec (Notion)
Fetched: Auth Design (Notion)
Fetched: 3 files from ./legacy-docs/payments/
Created: fab/memory/api/endpoints.md
Created: fab/memory/api/authentication.md
Created: fab/memory/payments/checkout.md
Created: fab/memory/payments/refunds.md
Updated: fab/memory/api/index.md
Updated: fab/memory/payments/index.md
Updated: fab/memory/index.md
Hydration complete — 4 files created, 0 updated.
```

### Re-hydration (same source)

```
Hydrating memory from 1 source(s)...
Fetched: API Spec (Notion)
Updated: fab/memory/api/endpoints.md (merged new content)
Updated: fab/memory/api/index.md
Updated: fab/memory/index.md
Hydration complete — 0 files created, 1 updated.
```

### Successful Generation

```
Scanning codebase for memory gaps...
Scanned 12 directories, 47 files.

## Memory Gap Report

### Modules
1. [High] auth module — src/auth/
2. [Medium] utils — src/utils/

### APIs
3. [High] REST API endpoints — src/api/routes/

Found 3 memory gaps.

User selects: "All"

Generating memory files...
Created: fab/memory/auth/index.md
Created: fab/memory/auth/auth-module.md
Created: fab/memory/api/endpoints.md
Updated: fab/memory/api/index.md
Updated: fab/memory/index.md
Generation complete — 2 files created, 1 domain index created, top-level index updated.
```

### Zero Gaps Found

```
Scanning codebase for documentation gaps...
Scanned 12 directories, 47 files.

No memory gaps found. fab/memory/ is up to date.
```

### Re-generation (idempotent merge)

```
Scanning codebase for memory gaps...
Scanned 12 directories, 47 files.

## Memory Gap Report

### APIs
1. [High] New webhook endpoints — src/api/webhooks/

Found 1 undocumented area. Document it?

Generating memory files...
Updated: fab/memory/api/endpoints.md (merged new content)
Updated: fab/memory/api/index.md
Updated: fab/memory/index.md
Generation complete — 0 files created, 1 updated.
```

### Scoped Scan (folder argument)

```
Scanning src/api/ for memory gaps...
Scanned 3 directories, 15 files.

## Memory Gap Report

### APIs
1. [High] REST API endpoints — src/api/routes/
2. [Medium] API middleware — src/api/middleware/

Found 2 undocumented areas. Document both?

Generating memory files...
Created: fab/memory/api/endpoints.md
Created: fab/memory/api/middleware.md
Updated: fab/memory/api/index.md
Updated: fab/memory/index.md
Generation complete — 2 files created.
```

---

## Idempotency Guarantee

This skill is safe to run any number of times with the same inputs.

### Ingest Mode

- **New files**: Created on first hydration, merged on subsequent runs
- **Existing files**: New requirements are added, existing requirements are updated if the source content changed, manually-added content is preserved
- **Domain indexes**: Updated with new entries, existing entries preserved
- **Top-level index**: Updated with new domains and expanded file lists, existing entries preserved
- **No deletions**: Hydration never removes existing content from memory files or indexes

### Generate Mode

- **Re-scan merges**: Running generate mode again merges new findings into existing generated memory files — does not overwrite
- **Manual edits preserved**: If a user edits a generated memory file (e.g., removes `[INFERRED]` markers, adds details), those edits are preserved on re-generation
- **New gaps appear**: Gaps introduced since the last run (new code, new modules) appear in the gap report
- **Covered areas excluded**: Previously covered areas do not appear as gaps (or appear with reduced priority)

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/memory/` missing | Abort with: "fab/memory/ not found. Run /fab-init first to create the memory directory." |
| `fab/memory/index.md` missing | Abort with same message (structural bootstrap incomplete) |
| No sources provided | Enter generate mode (scan from project root) |
| Mixed-mode arguments | Reject with: "Cannot mix ingest sources (URLs, .md files) with generate targets (folders). Run separately." |
| Folder path doesn't exist | Report error: "Folder not found: {path}" and abort |
| Zero gaps found | Report "No memory gaps found. fab/memory/ is up to date." and exit cleanly |
| Source URL unreachable | Report the error for that source, continue with remaining sources |
| Source content unreadable | Report the error, skip that source, continue |
| Domain folder already exists | Use it (do not recreate) |
| Memory file already exists | Merge new content into existing memory file |
| Domain index already exists | Update with new/modified entries |

---

Next: `/fab-new <description>`, `/fab-hydrate <more-sources>`, or `/fab-hydrate` (generate mode)
