---
name: fab-hydrate
description: "Hydrate docs from external sources or generate from codebase analysis. Safe to re-run."
---

# /fab-hydrate [sources...|folders...]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Hydrate `fab/docs/` from external sources or from codebase analysis.

- **Ingest mode** (URLs, `.md` files): Fetches or reads the provided sources, analyzes content to identify domains and topics, creates or merges documentation files, and maintains indexes.
- **Generate mode** (folders, no arguments): Scans the codebase for undocumented areas, presents an interactive gap report, lets the user select which gaps to document, and generates structured docs into `fab/docs/`.

Mode is determined automatically by argument type — no flags needed. Safe to run repeatedly — content is merged into existing docs without duplication or overwriting manually-added content.

---

## Pre-flight Check

Before doing anything else:

1. Check that `fab/docs/` directory exists
2. Check that `fab/docs/index.md` exists and is readable

**If either check fails, STOP immediately.** Output this message and do nothing else:

> `fab/docs/ not found. Run /fab-init first to create the docs directory.`

Do NOT create `fab/docs/` or `fab/docs/index.md`. The structural bootstrap (`/fab-init`) must have run first.

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
3. Map each topic to a target file: `fab/docs/{domain}/{topic}.md`

### Step 3: Create or Merge Documentation

For each identified topic:

1. If `fab/docs/{domain}/` directory does not exist, create it
2. If `fab/docs/{domain}/index.md` does not exist, create a domain index following the [Domain Index format](../../specs/templates.md#domain-index-fabdocsdomainindexmd):
   ```markdown
   # {Domain} Documentation

   | Doc | Description | Last Updated |
   |-----|-------------|-------------|
   ```
3. If the target doc file does not exist, create it with the analyzed content, structured as a [Centralized Doc](../../specs/templates.md#individual-doc-fabdomainnamemd) (with Overview, Requirements sections, Design Decisions, Changelog)
4. If the target doc file already exists, **merge** the new content into the existing doc — add new requirements, update existing ones, do not remove existing content. Preserve any manually-added content.

### Step 4: Update Domain Indexes

For each domain that was created or had docs added:

1. Read `fab/docs/{domain}/index.md`
2. Add rows for any new docs that were created in this domain
3. Update "Last Updated" column for any docs that were modified
4. Use relative links: `| [{name}]({name}.md) | {description} | {DATE} |`

### Step 5: Update Top-Level Index

After all sources are processed:

1. Read the current `fab/docs/index.md`
2. Add rows for any new domains that were created: `| [{domain}]({domain}/index.md) | {description} | {doc-list} |`
3. Update the doc-list column for existing domains that had new docs added (comma-separated list of all docs in the domain)
4. Do not remove existing entries
5. Use relative links (not absolute paths) — see [Top-Level Index format](../../specs/templates.md#top-level-index-fabdocsindexmd)

Report what was created and updated:
```
Created: fab/docs/{domain}/{topic}.md
Updated: fab/docs/{domain}/index.md
Updated: fab/docs/index.md
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

1. **Modules**: Enumerate top-level source directories. For each, check if a matching domain exists in `fab/docs/index.md`. Unmatched directories → Module gap.

2. **APIs**: Grep for route definitions, endpoint handlers, CLI command registrations, and exported public interfaces. Look for patterns like `app.get(`, `@route`, `export function`, `def command`, etc. Cross-reference against existing `fab/docs/` entries. Undocumented endpoints/exports → API gap.

3. **Patterns**: Identify recurring structural patterns across the codebase — middleware chains, plugin directories, event handler registrations, factory functions, decorator usage. If a pattern appears 3+ times and has no corresponding doc in `fab/docs/`, flag it as a Pattern gap.

4. **Configuration**: Glob for config files (`.env*`, `*.config.*`, `config/`, `settings.*`) and grep for environment variable references (`process.env`, `os.environ`, `ENV[]`). Undocumented config → Configuration gap.

5. **Conventions**: Analyze file naming patterns, directory structure conventions, common prefixes/suffixes. These are lowest priority and only flagged when the pattern is clear and consistent.

#### Cross-Reference Against Existing Docs

For each potential gap found, check `fab/docs/` domains and their doc entries:
- If the area is already covered by an existing doc → **exclude** it from the gap report (or deprioritize it)
- If the area is partially covered → include it with a note about what's missing

### Step 2: Gap Report & Interactive Scoping

#### Zero Gaps

If the scan finds no undocumented areas, output:

> `No documentation gaps found. fab/docs/ is up to date.`

Then stop — do not present the selection UI or proceed to doc generation.

#### Gap Report Format

Present a numbered gap report, grouped by category with priorities. Format:

```
## Documentation Gap Report

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

Parse the user's selection and proceed to doc generation with only the selected gaps.

#### Small Number of Gaps (1-3)

If 1-3 gaps are found, skip the AskUserQuestion UI. Instead, confirm:

> `Found {N} undocumented area(s). Document all?`

On confirmation, proceed to doc generation for all of them.

### Step 3: Doc Generation

For each selected gap, generate a documentation file.

#### Reading Source Code

Before generating a doc, read **all source files** within the gap's scope (directory, file set, or pattern matches). Synthesize the content into **one doc per gap** — not one doc per file. For example, a module gap covering `src/auth/` produces a single doc that covers the entire auth module, not separate docs for each file in the directory.

#### Doc Format

Write each generated doc to `fab/docs/{domain}/{topic}.md` using the centralized doc format:

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

After generating docs, maintain indexes using the **same logic as ingest mode Steps 4-5**:

1. Create or update `fab/docs/{domain}/index.md` for each domain touched
2. Update `fab/docs/index.md` with new domains and doc lists
3. All links SHALL be relative
4. Existing entries SHALL NOT be removed

This step is shared between ingest and generate modes.

---

## Output

### Successful Hydration

```
Hydrating docs from {N} source(s)...
Fetched: {title} ({source type})
Fetched: {N} files from {directory path}
Created: fab/docs/{domain}/{topic}.md
Created: fab/docs/{domain}/{topic}.md
Updated: fab/docs/{domain}/index.md
Updated: fab/docs/index.md
Hydration complete — {N} docs created, {M} updated.
```

### Multiple Sources

```
Hydrating docs from 3 source(s)...
Fetched: API Spec (Notion)
Fetched: Auth Design (Notion)
Fetched: 3 files from ./legacy-docs/payments/
Created: fab/docs/api/endpoints.md
Created: fab/docs/api/authentication.md
Created: fab/docs/payments/checkout.md
Created: fab/docs/payments/refunds.md
Updated: fab/docs/api/index.md
Updated: fab/docs/payments/index.md
Updated: fab/docs/index.md
Hydration complete — 4 docs created, 0 updated.
```

### Re-hydration (same source)

```
Hydrating docs from 1 source(s)...
Fetched: API Spec (Notion)
Updated: fab/docs/api/endpoints.md (merged new content)
Updated: fab/docs/api/index.md
Updated: fab/docs/index.md
Hydration complete — 0 docs created, 1 updated.
```

### Successful Generation

```
Scanning codebase for documentation gaps...
Scanned 12 directories, 47 files.

## Documentation Gap Report

### Modules
1. [High] auth module — src/auth/
2. [Medium] utils — src/utils/

### APIs
3. [High] REST API endpoints — src/api/routes/

Found 3 documentation gaps.

User selects: "All"

Generating docs...
Created: fab/docs/auth/index.md
Created: fab/docs/auth/auth-module.md
Created: fab/docs/api/endpoints.md
Updated: fab/docs/api/index.md
Updated: fab/docs/index.md
Generation complete — 2 docs created, 1 domain index created, top-level index updated.
```

### Zero Gaps Found

```
Scanning codebase for documentation gaps...
Scanned 12 directories, 47 files.

No documentation gaps found. fab/docs/ is up to date.
```

### Re-generation (idempotent merge)

```
Scanning codebase for documentation gaps...
Scanned 12 directories, 47 files.

## Documentation Gap Report

### APIs
1. [High] New webhook endpoints — src/api/webhooks/

Found 1 undocumented area. Document it?

Generating docs...
Updated: fab/docs/api/endpoints.md (merged new content)
Updated: fab/docs/api/index.md
Updated: fab/docs/index.md
Generation complete — 0 docs created, 1 updated.
```

### Scoped Scan (folder argument)

```
Scanning src/api/ for documentation gaps...
Scanned 3 directories, 15 files.

## Documentation Gap Report

### APIs
1. [High] REST API endpoints — src/api/routes/
2. [Medium] API middleware — src/api/middleware/

Found 2 undocumented areas. Document both?

Generating docs...
Created: fab/docs/api/endpoints.md
Created: fab/docs/api/middleware.md
Updated: fab/docs/api/index.md
Updated: fab/docs/index.md
Generation complete — 2 docs created.
```

---

## Idempotency Guarantee

This skill is safe to run any number of times with the same inputs.

### Ingest Mode

- **New docs**: Created on first hydration, merged on subsequent runs
- **Existing docs**: New requirements are added, existing requirements are updated if the source content changed, manually-added content is preserved
- **Domain indexes**: Updated with new entries, existing entries preserved
- **Top-level index**: Updated with new domains and expanded doc lists, existing entries preserved
- **No deletions**: Hydration never removes existing content from docs or indexes

### Generate Mode

- **Re-scan merges**: Running generate mode again merges new findings into existing generated docs — does not overwrite
- **Manual edits preserved**: If a user edits a generated doc (e.g., removes `[INFERRED]` markers, adds details), those edits are preserved on re-generation
- **New gaps appear**: Gaps introduced since the last run (new code, new modules) appear in the gap report
- **Documented areas excluded**: Previously documented areas do not appear as gaps (or appear with reduced priority)

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/docs/` missing | Abort with: "fab/docs/ not found. Run /fab-init first to create the docs directory." |
| `fab/docs/index.md` missing | Abort with same message (structural bootstrap incomplete) |
| No sources provided | Enter generate mode (scan from project root) |
| Mixed-mode arguments | Reject with: "Cannot mix ingest sources (URLs, .md files) with generate targets (folders). Run separately." |
| Folder path doesn't exist | Report error: "Folder not found: {path}" and abort |
| Zero gaps found | Report "No documentation gaps found. fab/docs/ is up to date." and exit cleanly |
| Source URL unreachable | Report the error for that source, continue with remaining sources |
| Source content unreadable | Report the error, skip that source, continue |
| Domain folder already exists | Use it (do not recreate) |
| Doc file already exists | Merge new content into existing doc |
| Domain index already exists | Update with new/modified entries |

---

Next: `/fab-new <description>`, `/fab-hydrate <more-sources>`, or `/fab-hydrate` (generate mode)
