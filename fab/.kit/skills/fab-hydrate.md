---
name: fab-hydrate
description: "Ingest external documentation into fab/docs/ with domain mapping and index maintenance. Safe to re-run."
---

# /fab:hydrate [sources...]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Ingest external documentation into `fab/docs/`. Fetches or reads the provided sources, analyzes content to identify domains and topics, creates or merges documentation files, and maintains both top-level and domain-level indexes. Safe to run repeatedly — content is merged into existing docs without duplication or overwriting manually-added content.

---

## Pre-flight Check

Before doing anything else:

1. Check that `fab/docs/` directory exists
2. Check that `fab/docs/index.md` exists and is readable

**If either check fails, STOP immediately.** Output this message and do nothing else:

> `fab/docs/ not found. Run /fab:init first to create the docs directory.`

Do NOT create `fab/docs/` or `fab/docs/index.md`. The structural bootstrap (`/fab:init`) must have run first.

---

## Arguments

- **`[sources...]`** *(required)* — one or more URLs or local paths containing documentation to ingest into `fab/docs/`. Supported source types:
  - **Notion URLs** — pages or databases (fetch via Notion MCP or API)
  - **Linear URLs** — issues or projects (fetch via Linear MCP or API)
  - **Local files/directories** — markdown, text, or directories of docs (read from filesystem; if directory, read all markdown files recursively)

**If no sources are provided, STOP.** Output:

> `Usage: /fab:hydrate <source> [source...]`
> `Provide one or more URLs or local paths to ingest.`

---

## Behavior

For each source provided:

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
2. If `fab/docs/{domain}/index.md` does not exist, create a domain index following the [Domain Index format](../../doc/fab-spec/TEMPLATES.md#domain-index-fabdocsdomainindexmd):
   ```markdown
   # {Domain} Documentation

   | Doc | Description | Last Updated |
   |-----|-------------|-------------|
   ```
3. If the target doc file does not exist, create it with the analyzed content, structured as a [Centralized Doc](../../doc/fab-spec/TEMPLATES.md#individual-doc-fabdomainnamemd) (with Overview, Requirements sections, Design Decisions, Changelog)
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
5. Use relative links (not absolute paths) — see [Top-Level Index format](../../doc/fab-spec/TEMPLATES.md#top-level-index-fabdocsindexmd)

Report what was created and updated:
```
Created: fab/docs/{domain}/{topic}.md
Updated: fab/docs/{domain}/index.md
Updated: fab/docs/index.md
```

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

---

## Idempotency Guarantee

This skill is safe to run any number of times with the same sources:

- **New docs**: Created on first hydration, merged on subsequent runs
- **Existing docs**: New requirements are added, existing requirements are updated if the source content changed, manually-added content is preserved
- **Domain indexes**: Updated with new entries, existing entries preserved
- **Top-level index**: Updated with new domains and expanded doc lists, existing entries preserved
- **No deletions**: Hydration never removes existing content from docs or indexes

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/docs/` missing | Abort with: "fab/docs/ not found. Run /fab:init first to create the docs directory." |
| `fab/docs/index.md` missing | Abort with same message (structural bootstrap incomplete) |
| No sources provided | Abort with usage message |
| Source URL unreachable | Report the error for that source, continue with remaining sources |
| Source content unreadable | Report the error, skip that source, continue |
| Domain folder already exists | Use it (do not recreate) |
| Doc file already exists | Merge new content into existing doc |
| Domain index already exists | Update with new/modified entries |

---

Next: `/fab:new <description>` or `/fab:hydrate <more-sources>`
