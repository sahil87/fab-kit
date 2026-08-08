---
name: docs-hydrate-memory
description: "Hydrate memory from external sources or generate from codebase analysis. Safe to re-run."
---

# /docs-hydrate-memory [sources...|folders...]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

## Contents

- Purpose
- Pre-flight Check
- Context Loading
- Arguments
- Ingest Mode Behavior
- Generate Mode Behavior
- Backfill Mode Behavior
- Output
- Idempotency
- Error Handling

---

## Purpose

Hydrate `docs/memory/` from external sources or from codebase analysis.

- **Ingest mode** (URLs, `.md` files): Fetches/reads sources, identifies domains and topics, creates or merges memory files, maintains indexes.
- **Generate mode** (folders, no arguments): Scans codebase for undocumented areas, presents interactive gap report, generates memory files.
- **Backfill mode** (`backfill` keyword, or dispatched by `/docs-reorg-memory`): Re-scans an existing `docs/memory/` tree for topic files that lack `description:` frontmatter and adds the FKF frontmatter (`type: memory` + `description:`) — **body-preserving** (only prepends/edits leading frontmatter; never strips an existing `## Changelog` body). Used to migrate a pre-fab-kit, hand-curated tree to the fab-kit convention so `fab memory-index` stops rendering `—` for every row. Unlike generate mode (which *creates* files from source-code gaps), backfill *adds frontmatter to existing* files.

Mode is determined automatically by argument type (ingest/generate) or by the explicit `backfill` keyword / reorg dispatch. Safe to run repeatedly — content is merged as current truth: the affected topic section is rewritten to current truth (not appended as a change-keyed delta), which neither duplicates existing entries nor overwrites manually-added content; backfill skips files that already have `description:`.

### Index Ownership

Index files (`index.md` at the root, domain, and sub-domain tiers) are **generated artifacts** — `fab memory-index` is their single writer. The one hand-curated field is the `description:` frontmatter (on topic files and on domain/sub-domain indexes). When a new domain or sub-domain is created, its `index.md` **stub** — only the `description:` frontmatter one-liner, nothing else — is created **before** `fab memory-index` runs; the command fills in the generated body and round-trips the description. Never hand-edit generated index rows. Both modes below follow this model.

> **Refuse-before-regen guard (destructive-loss).** Before any `fab memory-index` regeneration step below, consult `fab memory-index --check`: on **exit 2** (destructive loss — a curated description would regenerate to `—`, a tombstone row would drop, or a custom grouping would flatten), **refuse to regenerate** and surface the pointer `→ run /docs-reorg-memory to remediate (it relocates removal-history rows to _shared/removed-domains.md and backfills description: frontmatter via /docs-hydrate-memory) before regenerating.` (`/docs-reorg-memory` is the orchestrator for all three tier-2 categories — it relocates tombstone rows itself and dispatches *this* skill's backfill mode; backfill alone does NOT relocate tombstones.) **No-op on born-compatible fab-kit trees** (always exit 0/1, never 2 — not dead code); it fires only on a pre-fab-kit tree reached via ingest/generate before backfill. Backfill mode itself only adds frontmatter, never destroys, so by the time *it* regenerates the guard is already a no-op.

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

**Mode disambiguation** — backfill is checked first: it is reached only by the explicit `backfill` keyword or a reorg dispatch, so it never collides with bare ingest/generate routing. The two are otherwise distinct by intent: **generate** *creates* memory files from source-code gaps; **backfill** *adds `description:` frontmatter to existing* memory files (no new content). All non-backfill arguments must classify to the same mode — **mixed-mode → reject** (Error Handling).

**Backfill takes no extra arguments** — it is an independent re-scan of `docs/memory/` with no caller manifest (see Backfill Mode Step 1), so any positional argument after the `backfill` keyword is meaningless; `backfill` first **and** any further argument → **reject** (Error Handling). The reorg-dispatch form never supplies extra args — it names only the operation.

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
3. If target file doesn't exist → read `$(fab kit-path)/templates/memory.md`,
   create its full topic-file skeleton, and apply the FKF authoring rules below.
4. If target file exists → merge the affected section as current truth under the
   same rules. Preserve manual content, stamp a missing `type: memory`, and
   re-check that `description:` still routes after any body edit.

### FKF authoring rules

- Lead every created or updated topic file with `type: memory` and a one-line,
  change-id-free `description:` of at most 500 characters (FKF §3.1–§3.2).
  Detail and provenance belong in the body; never hand-write generated index rows.
- Write present truth only (FKF §3.3): remove superseded statements instead of
  narrating transitions; keep provenance as trailing citations or
  `*Introduced by*`; keep change IDs out of headings.
- Put durable rationale in a `## Design Decisions` entry using **Decision**,
  **Why**, **Rejected**, and *Introduced by*. Never use a changelog bullet there.
- Use the template's Overview / Requirements / Design Decisions skeleton for new
  files and omit `## Changelog`; folder `log.md` owns change history.
- Use bundle-relative `/...` links within memory; external links remain
  repo-relative or absolute.

**Shape bounds (SHOULD guidance)** when placing topics into domains:
- Aim for **~5–12 topic files per folder**. Past ~12, `fab memory-index` warns — consider a sub-domain.
- **Max depth 3**: `docs/memory/{domain}/{sub-domain}/{topic}.md`.
- Introduce a sub-domain **only reactively**, when a cohesive cluster of **≥8 files** exists. Never pre-build hierarchy.
- Reserved domains `_shared/` (cross-cutting) and `_unsorted/` (staging) are exempt from the width bound.

### Step 3.5: Post-Hydrate Self-Check (before regen)

Re-read every memory file you created or merged **this run** and strip any transition phrasing just introduced — no "renamed / now / previously / no longer / was `old.value`" narration, no change-keyed delta paragraph left below an older paragraph on the same topic, no change-ids in headings — and confirm each touched file's `description:` still routes (one line, ≤500 chars, change-id-free, FKF §3.2). This is a self-review of *this run's own writes*, **not** a corpus sweep (draining pre-existing debt across the tree is `/docs-distill-memory`'s job). A merge that already rewrote each section to current truth leaves nothing to strip — the check is the safety net for narration reflexively introduced during the write. (Generate mode runs the identical check on the files it generated.)

### Step 4: Regenerate Indexes (`fab memory-index`)

Run `fab memory-index` once to regenerate the root (domains-only), every domain index, and every sub-domain index from folder contents + `description:` frontmatter (the single writer — see Index Ownership; never hand-edit rows). On any merge conflict in a generated `docs/memory/**/index.md` or `log.md`, do **not** hand-merge: resolve the topic files, re-run `fab memory-index`, take its output wholesale (FKF §5). The index carries no dates — it is a pure function of content. Any non-fatal shape warnings it prints to stderr are advisory (over-wide / over-deep folders, and a 501–1000-char over-length `description:`); note the two `description:` escalations are **blocking**, not advisory — a change-id in `description:` and a gross over-cap value (> 1000 chars, 2× the 500 soft cap) fail `--check` (FKF §3.2).

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

For each selected gap, read all source files in scope and synthesize one memory
file per gap using ingest Step 3 and its FKF authoring rules. Derive RFC 2119
requirements from code, include Design Decisions where inferable, and strip the
template's guidance comments.

Mark ambiguous inferences with `[INFERRED]` inline near the relevant requirement.

**Placement** follows ingest-mode Step 3 exactly: target path `docs/memory/{domain}/{topic}.md` (or `.../{sub-domain}/{topic}.md`); create the domain folder and its `description:`-only index stub (sub-domain stub likewise) before Step 4 runs (see Index Ownership); and the same **Shape bounds** apply (see ingest Step 3).

### Step 3.5: Post-Hydrate Self-Check (before regen)

Run the same post-hydrate self-check as ingest Step 3.5, scoped to the files generated this run: re-read each generated file, strip any transition phrasing / change-keyed delta paragraph / change-id heading just introduced, and confirm each `description:` still routes (one line, ≤500 chars, change-id-free). A self-review of this run's own writes, not a corpus sweep.

### Step 4: Regenerate Indexes

Same as ingest mode Step 4 — run `fab memory-index` to regenerate the root (domains-only), domain, and sub-domain indexes from folder contents + frontmatter. Do not hand-edit index rows (and never hand-merge a generated index/log conflict — resolve topic files, re-run, take wholesale, FKF §5).

---

## Backfill Mode Behavior

Backfill migrates an **existing** hand-curated `docs/memory/` tree (typically pre-fab-kit) to the convention `fab memory-index` depends on: each topic file leads with a `description:` frontmatter line. Without it, the generator (which reads descriptions exclusively from frontmatter) renders `—` for every row, wiping curated descriptions on the first regen. Backfill is the one-time fix. It is invoked directly (`/docs-hydrate-memory backfill`) or dispatched by `/docs-reorg-memory` as the second step of its compatibility orchestration.

> **Scope**: Backfill is a **pure frontmatter operation** — it adds the FKF frontmatter (`type: memory` + `description:`) to existing files and creates missing `description:`-only index stubs. It does NOT detect or relocate tombstone rows, flatten custom groupings, move files, or strip existing `## Changelog` bodies; those structural concerns belong to `/docs-reorg-memory` (and the `## Changelog` strip to FKF migration Change 4). The body of every file is preserved byte-for-byte. **Backfill is exempt from the ingest/generate Step 3.5 post-hydrate self-check** — that step edits bodies (stripping transition phrasing), which the body-preserving contract forbids; backfill applies only the change-id-free `description:` rule of FKF §3.2 (Step 2), never the body-style rules of §3.3.

### Step 1: Re-scan `docs/memory/` (no caller manifest)

Backfill **walks `docs/memory/` itself** to find every topic file (a non-`index.md` `.md` file) lacking a `description:` frontmatter field — it does **not** receive a file list from its caller. This holds for both forms: the direct-user invocation and the reorg dispatch (reorg's prompt names the operation — "backfill this tree" — not the files). A file with no frontmatter, or frontmatter without a `description:` key, counts as missing (the same `frontmatter.Field` semantics `fab memory-index` uses). The walk is the loose, idempotent seam between the two independently-invocable skills.

### Step 2: Synthesize and write `description:` frontmatter (body-preserving)

For each discovered topic file missing `description:`:

1. Read the file's **own content** — Overview, first section, or `# H1` — and synthesize a concise one-line summary. Keep it a **one-liner capped at 500 characters and free of change-ids** (FKF §3.2) — a routing signal, not a summary of the file or a provenance record.
2. **Prefer a curated index row** where one maps to this file. If an existing hand-curated index file (e.g., a pre-fab-kit `index.md` whose rows line up file-by-file with the topic files) has a row whose description text describes this file, use that curated text as the source — it is higher quality than re-synthesis. When reusing such pre-fab-kit row text, **strip any change-ids** it carries (a trailing `— xu0k`-style suffix or a `(d9rs)`-style citation) so the resulting `description:` stays change-id-free per item 1 (FKF §3.2).
3. Write the FKF frontmatter (`type: memory` + synthesized `description:`, per ingest Step 3) as the **leading frontmatter block** — but take the **frontmatter shape only** from `$(fab kit-path)/templates/memory.md`, **never its body skeleton.** Backfill is a pure-frontmatter operation: **preserve the existing body byte-for-byte** — only prepend/edit the leading frontmatter, never the content below, never impose the template's Overview/Requirements/Design Decisions skeleton. In particular it does **NOT** strip an existing `## Changelog` section; see `$(fab kit-path)/reference/fkf.md` for the shipped FKF contract.
4. **Skip files that already have a `description:`** — backfill never overwrites an existing one (and stamps `type: memory` only when it is adding the frontmatter for the first time). This makes a second pass a no-op (idempotency — a fab-kit design principle).

### Step 3: Create missing index stubs (stub-before-index)

For any domain/sub-domain folder lacking an `index.md` (or whose `index.md` lacks `description:`), create the `description:`-only `index.md` **stub** the same way ingest/generate modes do — only the `description:` frontmatter one-liner, nothing else, created **before** any index regeneration (see Index Ownership above). This gives `fab memory-index` the domain description to read.

### Step 4: Caller-aware index regeneration

Backfill is **caller-aware** about `fab memory-index`:

- **Dispatched by `/docs-reorg-memory`** (the dispatch prompt carries the reorg-dispatched / defer-regen signal): do **NOT** run `fab memory-index`. reorg runs it exactly once at the end of its orchestration (after rebalance), so a regen here would be redundant work and would race reorg's single regen.
- **Invoked directly by a user** (no reorg signal): run `fab memory-index` as the final step, exactly like ingest and generate modes — root (domains-only) + every domain + every sub-domain index, regenerated from folder contents + frontmatter.

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
